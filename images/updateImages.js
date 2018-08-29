// This script will check the database for original images and upload optimized ones
let fs = require('fs');
let crypto = require('crypto');

let pg = require('pg');
let config = require('config');
let winston = require('winston');
let AWS = require('aws-sdk');
let axios = require('axios');
const imagemin = require('imagemin');
const sharp = require('sharp');
const imageminMozjpeg = require('imagemin-mozjpeg');
const imageminPngquant = require('imagemin-pngquant');

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
            (err, res) => {
                if (err) {
                    logger.error(err);
                }
                for (let imgObj of res.rows) {
                    this.minifyImage(imgObj);
                    break;
                }
            }
        )
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
        axios({
            method: 'get',
            url: imageObj.imageurl,
            responseType: 'stream'
        }).then((response) => {
            return new Promise((resolve, reject) => {
                const file = response.data.pipe(fs.createWriteStream(`images/imagesToBeProcessed/${digest}.jpg`));
                file.on("finish", () => { resolve(); }); // not sure why you want to pass a boolean
                file.on("error", reject);
            })
        }).catch(err => {
            console.log(err)
        }).then(() => {
            // Resize the image
            return sharp(`images/imagesToBeProcessed/${digest}.jpg`)
                .resize(520)
                .toFile(`images/resized/${digest}.jpg`)
        }).then(() => {
            // Compress the image
            return imagemin(['./images/resized/*.{jpg,png}'], './images/imagesToBeProcessed', {
                plugins: [
                    imageminMozjpeg({ quality: '90', progressive: true }),
                    imageminPngquant({ speed: 1, quality: '65-80' })
                ],
            });
        }).then((files) => {
            console.log(files);
        }).catch(err => {
            console.log(err);
        })
    }
}

let podcast = new Podcast()
podcast.createNewImages();
