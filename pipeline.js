#!/usr/bin/env node

// https://docs.aws.amazon.com/AmazonECR/latest/public/push-oci-artifact.html

const execa = require('execa')
const semanticRelease = require('semantic-release')
const Docker = require('dockerode');
const path = require("path");
const ghpages = require('gh-pages');
const util = require('util');
const exec = util.promisify(require('child_process').exec);
const yaml = require('js-yaml');
const https = require('https');
const fs = require("fs");
const fsPromises = require('fs').promises;

const {DOCKER_REPOSITORY} = process.env

const docker = new Docker({socketPath: '/var/run/docker.sock'});

const workloads = [
    {
        name: 'saml-proxy',
        path: '',
        imageName: 'saml-proxy'
    },
]

async function getLatestVersion() {
    const {stdout} = await execa('git', ['tag', '--points-at', 'HEAD'])
    return stdout
}

console.log("Environment: ", process.env.ENV)

async function release() {
    const result = await semanticRelease({
        branches: [
            'master',
        ],
        repositoryUrl: process.env.GITHUB_REPOSITORY_URL,
        dryRun: process.env.ENV === 'dev',
        ci: true,
        npmPublish: false
    }, {
        env: {
            ...process.env,
            GITHUB_TOKEN: process.env.GITHUB_TOKEN,
            GIT_AUTHOR_NAME: 'MatteoGioioso'
        },
    });

    let version;

    if (result) {
        const {lastRelease, commits, nextRelease, releases} = result;
        version = nextRelease.version

        console.log(`Published ${nextRelease.type} release version ${nextRelease.version} containing ${commits.length} commits.`);

        if (lastRelease.version) {
            console.log(`The last release was "${lastRelease.version}".`);
        }

        for (const release of releases) {
            console.log(`The release was published with plugin "${release.pluginName}".`);
        }
    } else {
        const tag = await getLatestVersion()
        const version = tag.replace('v', '')
        console.log(`No release published. Current version: ${version}, DONE`);
        return undefined
    }

    return version
}

function getImageFullname(version) {
    return `${DOCKER_REPOSITORY}:${version}`
}

async function dockerLogin() {
    return {
        auth: "",
        username: process.env.DOCKER_USERNAME,
        password: process.env.DOCKER_ACCESS_TOKEN,
        serveraddress: 'https://index.docker.io/v1'
    };
}

async function dockerBuild(version) {
    const image = getImageFullname(version)
    console.log("Building image:", image)

    const stream = await docker.buildImage(
        {context: __dirname, src: ['Dockerfile', '/src']},
        {t: image},
    )

    await attachSTOUTtoStream(stream)
    console.log(`Building saml-proxy DONE`)
}


async function dockerTag(imageObj, version) {
    const imageFullName = getImageFullname(version)

    console.log("Tagging image:", DOCKER_REPOSITORY, "==>", version)

    await imageObj.tag({
        name: imageFullName,
        repo: DOCKER_REPOSITORY,
        tag: version
    })
}

async function pushImage(imageObj, version, auth) {
    const imageFullName = getImageFullname(version)
    const stream = await imageObj.push({name: imageFullName, authconfig: auth});
    await attachSTOUTtoStream(stream)
    console.log(`Pushing ${imageFullName} DONE`)
}

async function attachSTOUTtoStream(stream) {
    await new Promise((resolve, reject) => {
        const pipe = stream.pipe(process.stdout, {
            end: true
        });

        pipe.on('end', () => {
            resolve()
        })

        pipe.on('error', (err) => {
            reject(err)
        })

        stream.on('error', (err) => {
            reject(err)
        })

        stream.on('end', () => {
            resolve()
        })
    })
}

/**
 * Pushing the new version of the chart will delete the old version,
 * therefore we need to download all the previous charts.
 */
async function downloadPreviousHelmReleases() {
    // List all the files that are on the gh-pages branch
    const {stdout, stderr} = await exec(
        "git ls-tree origin/gh-pages -r --name-only"
    );
    console.log(stdout);
    console.log(stderr);
    // Files list is separated by a new line
    const files = [...stdout.split("\n")];
    console.log(files)

    const createDownload = async (fileName) => new Promise((resolve, reject) => {
        const file = fs.createWriteStream(`docs/${fileName}`);
        https.get(`https://raw.githubusercontent.com/MatteoGioioso/saml-proxy/gh-pages/${fileName}`, function (response) {
            response.pipe(file);
            file.on("finish", () => {
                file.close()
                resolve();
            });
            file.on("error", reject);
        });
    });

    for (const file of files) {
        if (!file || file === '') continue
        if (file.includes('saml-proxy')) {
            await createDownload(file.trim())
        }
    }
}

async function helm(version) {
    const chartDomain = "https://matteogioioso.github.io/saml-proxy/"
    const helmChartPath = "charts/saml-proxy"
    // version chart
    console.log("Versioning helm chart")
    const filePath = path.join(__dirname, `${helmChartPath}/Chart.yaml`)
    const valuesYaml = await fsPromises.readFile(filePath, 'utf8');
    const values = yaml.load(valuesYaml);
    values.version = version
    const newValues = yaml.dump(values)
    await fsPromises.writeFile(filePath, newValues)

    // dockerBuild chart: helm dependency update && helm package . -n rebugit
    console.log("Updating, packaging and creating chart index")
    const {stdout, stderr} = await exec(
        `helm dependency update ${helmChartPath}/ && helm package ${helmChartPath}/ -d docs/ && helm repo index docs --url ${chartDomain}`
    );
    console.log(stdout);
    console.log(stderr);
}

const prepareGhPagesPublishing = async () => {
    console.log("Copy files to docs/ folder")
    await fsPromises.mkdir('docs/assets', {recursive: true});
    await fsPromises.copyFile(path.join(__dirname, "README.md"), path.join(__dirname, "docs/README.md"))
    const src = `assets`;
    const dist = `docs/assets`;
    const {stdout, stderr} = await exec(`cp -r ${src}/* ${dist}`)
    console.log(stdout);
    console.log(stderr);
}

const publishing = () => new Promise((resolve, reject) => {
    // Push everything with ghpages
    console.log("Deploying to github")
    ghpages.publish(
        'docs',
        {
            repo: `https://git:${process.env.GITHUB_TOKEN}@github.com/${process.env.GITHUB_REPOSITORY_URL.replace("https://github.com/", "")}`,
            user: {
                name: process.env.GITHUB_USERNAME,
                email: process.env.GITHUB_EMAIL
            }
        },
        function (err) {
            if (err) {
                return reject(err)
            }

            console.log("Chart has been published")
            return resolve()
        }
    );
})

async function pipeline() {
    const version = await release();
    if (!version) {
        return
    }

    const auth = await dockerLogin();

    await dockerBuild(version);

    for (const v of [version, 'latest']) {
        const imageObjBeforeTag = await docker.getImage(getImageFullname(version));
        await dockerTag(imageObjBeforeTag, v);
        const imageObj = await docker.getImage(getImageFullname(v));
        await pushImage(imageObj, v, auth)
    }

    await prepareGhPagesPublishing()
    await downloadPreviousHelmReleases()
    await helm(version)
    await publishing()
}

pipeline().then().catch(e => {
    console.error(e)
    process.exit(1)
})