#!/usr/bin/env node

// https://docs.aws.amazon.com/AmazonECR/latest/public/push-oci-artifact.html

const execa = require('execa')
const semanticRelease = require('semantic-release')
const Docker = require('dockerode');
const path = require("path");
const AWS = require("aws-sdk");
const ghpages = require('gh-pages');
const util = require('util');
const exec = util.promisify(require('child_process').exec);
const yaml = require('js-yaml');
const https = require('https');
const fs = require("fs");
const fsPromises = require('fs').promises;


const docker = new Docker({socketPath: '/var/run/docker.sock'});
const ecr = new AWS.ECRPUBLIC({
  region: process.env.AWS_REGION,
  apiVersion: '2020-10-30',
  endpoint: new AWS.Endpoint(`api.ecr.ap-southeast-1.amazonaws.com`),
  credentials: {
    accessKeyId: process.env.AWS_ACCESS_KEY_ID,
    secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY,
    sessionToken: process.env.AWS_SESSION_TOKEN
  }
})

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

async function build(version, workload) {
  const image = `${workload.imageName}:${version}`
  const stream = await docker.buildImage(
    {context: path.join(__dirname, workload.path)},
    {t: image},
  )

  await attachSTOUTtoStream(stream)
  console.log(`Building ${workload.imageName} DONE`)
  return image
}


async function dockerLogin() {
  console.log("Get auth token from ecr public...")
  const login = await ecr.getAuthorizationToken().promise();

  return {
    auth: "",
    username: 'AWS',
    password: Buffer
      .from(login.authorizationData.authorizationToken, 'base64')
      .toString('utf-8')
      .replace('AWS:', ''),
    serveraddress: process.env.ECR_REPOSITORY_URL
  };
}

async function tagImage(imageToTag, tag) {
  const imageName = imageToTag.split(':')[0]
  const repo = process.env.ECR_REPOSITORY_URL

  const taggedImage = `${repo}/${imageName}:${tag}`
  const image = await docker.getImage(imageToTag);

  console.log("Tagging image:", imageToTag, "==>", taggedImage)

  await image.tag({
    name: imageToTag,
    repo: `${repo}/${imageName}`,
    tag
  })

  const taggedImageObj = await docker.getImage(taggedImage);

  return {
    taggedImageObj,
    taggedImage
  }
}

async function pushImage(imageObj, imageFullName, auth) {
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

  // build chart: helm dependency update && helm package . -n rebugit
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
  const version = '1.0.1'//await release();
  if (!version) {
    return
  }

  const auth = await dockerLogin();

  for (const workloadsKey in workloads) {
    const workload = workloads[workloadsKey]
    const imageFullName = await build(version, workload);
    const image = await tagImage(imageFullName, version);
    const imageLatest = await tagImage(imageFullName, "latest");
    await pushImage(image.taggedImageObj, image.taggedImage, auth)
    await pushImage(imageLatest.taggedImageObj, imageLatest.taggedImage, auth)
  }

  // await prepareGhPagesPublishing()
  // await downloadPreviousHelmReleases()
  // await helm(version)
  // await publishing()
}

pipeline().then().catch(e => {
  console.error(e)
  process.exit(1)
})