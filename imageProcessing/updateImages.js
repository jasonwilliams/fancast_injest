// This script will check the database for original images and upload optimized ones
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

/**
 * Class representing a podcast image to upload
 * @class Podcast
 */
class Podcast {
  constructor() {
    // Considering many network requests will be made for the same image, we should keep a mapping of images
    // we've already found, to save bandwidth
    this.checkedImages = new Map();

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

  /**s
   * createNewImages -  Creates images by checking the DB for images which haven't been processsed
   * @param {boolean} type true "podcast", false "podcast_episode"
   */
  createNewImages(type = true) {
    this.table = (type) ? 'podcasts' : 'podcast_episodes';
    return this.pool.query(`select image->'url' as imageUrl, id from ${this.table} where image->'ext' IS NULL AND image->'url' IS NOT NULL`)
      .then(async (res) => {
        for (let imgObj of res.rows) {
          logger.info(`processing ${imgObj.id}`);
          // e.g b15166fcba82035fed04.png?v=63688781273 remove query strings
          let ext = path.extname(imgObj.imageurl).replace(/\?.*$/, '') || '.png';
          try {
            let digest = this.createDigest(imgObj);
            let exists = await this.existsOnBucket(digest);
            // If image exists, we don't need to minify or upload
            if (!exists) {
              let imageBuffers = await this.minifyImage(imgObj, ext, digest);
              await this.uploadFiles(imageBuffers, digest, '.png');
            }
            await this.entryInDB(imgObj.id, digest, '.png');
          } catch (e) {
            logger.error(e.message);
          }
        }
        return true;
      })
      .catch(err => {
        console.log(err);
      })
  }

  /**
   * Check image exists using a digest
   * @param {string} digest - Used to check if an image already exists in the bucket
   * @returns {boolean} ifExists
   */
  existsOnBucket(digest) {
    let promise = new Promise((resolve, reject) => {
      // Search within checkedImages first!
      if (this.checkedImages.has(digest)) {
        resolve(this.checkedImages.get(digest));
        return;
      }

      s3.headObject({
        Bucket: 'fancast',
        Key: `podcast-images/${digest}.webp`,
      }, (err, data) => {
        if (err) {
          // a 404 is OK, this tell us the image does not exist
          if (err.statusCode === 404) {
            resolve(false);
          } else {
            reject(err);
            logger.error(err);
          }
          this.checkedImages.set(digest, false);
        } else {
          resolve(true)
          logger.info('existsOnBucket resolved');
          logger.info(data);
          this.checkedImages.set(digest, true);
        }
      })

    });

    return promise;
  }

  /**
   *
   * @param {object} The URL and name of the image to minify
   * @returns {object} The Object {name, body} of the image
   */
  minifyImage(imageObj, ext = '.png', digest) {
    // First fetch the image
    return axios({
      method: 'get',
      url: imageObj.imageurl,
      responseType: 'arraybuffer'
    }).then((response) => {
      return Buffer.from(response.data);
    }).then((buffer) => {
      // Resize the image
      let w520 = sharp(buffer)
        .resize(520)
        .png() // This helps with image quality a lot
        .toBuffer();
      let w320 = sharp(buffer)
        .resize(320)
        .png() // This helps with image quality a lot
        .toBuffer();
      return Promise.all([w520, w320]);
    }).then((sizes) => {
      // Lets set up promises
      let promises = [];
      // 520
      promises.push(imagemin.buffer(sizes[0], {
        plugins: [
          imageminMozjpeg({ quality: '90', progressive: true }),
          imageminPngquant({ speed: 1 })
        ],
      }))
      // 320
      promises.push(imagemin.buffer(sizes[1], {
        plugins: [
          imageminMozjpeg({ quality: '90', progressive: true }),
          imageminPngquant({ speed: 1 })
        ],
      }))

      // 520 webp
      promises.push(imagemin.buffer(sizes[0], {
        plugins: [
          imageminWebp()
        ],
      }));

      // 320 webp
      promises.push(imagemin.buffer(sizes[1], {
        plugins: [
          imageminWebp()
        ],
      }));

      return Promise.all(promises);
    })
  }


  /**
   * @param {object} imageObj - The image to create a hash from
   * @returns {string} The digest returned from hashing the image URL
   */
  createDigest(imageObj) {
    // Get hash from URL and use this as our digest
    const hash = crypto.createHash('sha256');
    hash.update(imageObj.imageurl);
    let digest = hash.digest("hex").substring(0, 20);
    return digest;
  }

  entryInDB(id, digest, ext) {
    ext = ext.replace('.', ''); // don't bother with the .
    let updateID = `update ${this.table} SET image = jsonb_set(image, '{id}', to_jsonb($1::text)) where id = $2;`
    let updateIDValues = [digest, id];

    let updateExt = `update ${this.table} SET image = jsonb_set(image, '{ext}', to_jsonb($1::text)) where id = $2;`
    let updateExtValues = [ext, id];
    return this.pool.query(updateID, updateIDValues)
      .then(() => {
        return this.pool.query(updateExt, updateExtValues)
      }, (err) => {
        console.log(err);
      })
  }

  uploadFiles(buffers, digest, ext) {
    let promises = [`${digest}--520w${ext}`, `${digest}--320w${ext}`, `${digest}--520w.webp`, `${digest}--320w.webp`].map((v, i) => {
      return new Promise((resolve, reject) => {
        let data = buffers[i];

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
            console.log(err.stack);
            reject(err);
          }
          resolve();
        });
      });
    });

    return Promise.all(promises);
  }
}

async function main() {
  let podcast = new Podcast();
  await podcast.createNewImages(true);
  console.log('------ Episodes ------');

  // Do the same again for episodes
  await podcast.createNewImages(false);
}

main().then(() => {
  process.exit(0);
});
