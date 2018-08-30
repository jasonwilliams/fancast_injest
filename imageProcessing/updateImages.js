// This script will check the database for original images and upload optimized ones
let fs = require('fs');
let crypto = require('crypto');
let path = require('path');

let pg = require('pg');
let config = require('config');
let winston = require('winston');
let AWS = require('aws-sdk');
let axios = require('axios');
const imagemin = require('imagemin');
const sharp = require('sharp');
const imageminMozjpeg = require('imagemin-mozjpeg');
const imageminPngquant = require('imagemin-pngquant');
const imageminWebp = require('imagemin-webp');

// Setup logging
const logger = winston.createLogger({
  level: 'info',
  format: winston.format.json(),
  transports: [
    //
    // - Write to all logs with level `info` and below to `combined.log`
    // - Write all logs error (and below) to `error.log`.
    //
    new winston.transports.File({ filename: 'error.log', level: 'error' }),
    new winston.transports.File({ filename: 'combined.log' })
  ]
});

//
// If we're not in production then log to the `console` with the format:
// `${info.level}: ${info.message} JSON.stringify({ ...rest }) `
//
if (process.env.NODE_ENV !== 'production') {
  logger.add(new winston.transports.Console({
    format: winston.format.simple()
  }));
}

// Configure database setup
const dbUser = config.get("database.user");
const dbPass = config.get("database.password");
const dbHost = config.get("database.host");
const dbName = config.get("database.database");
const connectionString = `postgres://${dbUser}:${dbPass}@${dbHost}:5432/${dbName}`;

// Configure Spaces
const spacesKeyID = config.get('spaces.key');
const spacesAccessKey = config.get('spaces.AccessKey');
const spacesEndpoint = new AWS.Endpoint('ams3.digitaloceanspaces.com')
let s3 = new AWS.S3({
  endpoint: spacesEndpoint,
  accessKeyId: spacesKeyID,
  secretAccessKey: spacesAccessKey
});

class Podcast {
  constructor() {
    this.pool = new pg.Pool({
      connectionString: connectionString
    })
    this.pool
      .connect()
      .then(() => logger.info("Postgresql Connected"))
      .catch(e => logger.error("connection error", e.stack));

    this.pool.on("error", err => {
      logger.error("models/podcast: Database Error");
      logger.error("Connection most likely reset on database side..");
      if (err) logger.error(err);
    });
  }

  createNewImages() {
    this.pool.query(
      "select image->'url' as imageUrl, id from podcasts where image->'optimisedUrl' IS NULL AND image->'url' IS NOT NULL",
      async (err, res) => {
        if (err) {
          logger.error(err);
        }
        for (let imgObj of res.rows) {
          let ext = path.extname(imgObj.imageurl);
          let digest = await this.minifyImage(imgObj);
          this.uploadFiles(digest, ext);
          await this.entryInDB(imgObj.id, digest, ext);
        }
      }
    )
  }

  entryInDB(id, digest, ext) {
    ext = ext.replace('.', ''); // don't bother with the .
    let updateID = "update podcasts SET image = jsonb_set(image, '{id}', \'\"$1::text\"\') where id = $2;"
    let updateIDValues = [digest, id];

    let updateExt = "update podcasts SET image = jsonb_set(image, '{ext}', \"$1::text\") where id = $2;"
    let updateExtValues = [ext, id];
    let promise = new Promise((resolve, reject) => {
      this.pool.query(updateID, updateIDValues)
        .then(() => {
          return this.pool.query(updateExt, updateExtValues)
        }, (err) => {
          console.log(err);
        })
        .then(() => {
          console.log(id);
          resolve();
        })
        .catch((err) => {
          console.log(err);
          reject(err)
        })
    })

    return promise;
  }

  uploadFiles(digest, ext) {
    [`${digest}--320w${ext}`, `${digest}--520w${ext}`, `${digest}--320w.webp`, `${digest}--520w.webp`].forEach(v => {
      fs.readFile(`imageProcessing/processed/${v}`, function (err, data) {
        if (err) {
          throw err;
        }

        let contentType;
        let newExt = path.extname(v);
        switch (newExt) {
          case '.jpg':
            contentType = 'image/jpeg';
            break;
          case '.png':
            contentType = 'image/png';
            break;
          case '.webp':
            contentType = 'image/webp';
            break;
        }


        s3.putObject({
          Bucket: 'fancast',
          Key: `podcast-images/${v}`,
          Body: data,
          ACL: 'public-read',
          CacheControl: 'public, max-age=31536000, immutable',
          ContentType: contentType
        }, function (err) {
          if (err) {
            console.log(err);
          }
        })
      })
    })
  }

  /**
   *
   * @param {object} The URL and name of the image to minify
   * @returns {object} The Object {name, body} of the image
   */
  minifyImage(imageObj) {
    // Get hash from URL and use this as our digest
    const hash = crypto.createHash('sha256');
    hash.update(imageObj.imageurl);
    let digest = hash.digest("hex").substring(0, 20);

    // First fetch the image
    return axios({
      method: 'get',
      url: imageObj.imageurl,
      responseType: 'stream'
    }).then((response) => {
      return new Promise((resolve, reject) => {
        const file = response.data.pipe(fs.createWriteStream(`imageProcessing/imagesToBeProcessed/${digest}.jpg`));
        file.on("finish", () => { resolve(); }); // not sure why you want to pass a boolean
        file.on("error", reject);
      })
    }).catch(err => {
      console.log(err)
    }).then(() => {
      // Resize the image
      let w520 = sharp(`imageProcessing/imagesToBeProcessed/${digest}.jpg`)
        .resize(520)
        .png() // This helps with image quality a lot
        .toFile(`imageProcessing/resized/${digest}--520w.jpg`);
      let w320 = sharp(`imageProcessing/imagesToBeProcessed/${digest}.jpg`)
        .resize(320)
        .png() // This helps with image quality a lot
        .toFile(`imageProcessing/resized/${digest}--320w.jpg`);
      return Promise.all([w520, w320]);
    }).then(() => {
      // Compress the image
      return imagemin([`./imageProcessing/resized/*${digest}*.{jpg,png}`], './imageProcessing/processed', {
        plugins: [
          imageminMozjpeg({ quality: '90', progressive: true }),
          imageminPngquant({ speed: 1, quality: '65-80' })
        ],
      });
    }).then(() => {
      return imagemin([`./imageProcessing/resized/*${digest}*.{jpg,png}`], './imageProcessing/processed', {
        plugins: [
          imageminWebp({ quality: '80' })
        ],
      });
    }).then(() => {
      return digest;
    }).catch(err => {
      console.log(err);
    })
  }
}

let podcast = new Podcast()
podcast.createNewImages();
