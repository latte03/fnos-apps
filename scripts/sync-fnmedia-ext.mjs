#!/usr/bin/env node
/**
 * 跟随 yujian0815/fnmedia-ext 的最新 Release，更新 fnpack.json。
 *
 * 该上游同时发布完整版与公开精简版；脚本默认读取完整版（-full），
 * 从包内 manifest 与 config/privilege 读取安装元数据。
 *
 * 用法：
 *   node scripts/sync-fnmedia-ext.mjs
 *   node scripts/sync-fnmedia-ext.mjs --slim
 *   node scripts/sync-fnmedia-ext.mjs --dry-run
 *   node scripts/sync-fnmedia-ext.mjs --tag v0.2.0
 *   node scripts/sync-fnmedia-ext.mjs --fpk ../.tmp/fnmedia-ext-0.2.0-full.fpk
 *   node scripts/sync-fnmedia-ext.mjs --proxy https://gh-proxy.com/
 *
 * 环境变量：
 *   GITHUB_TOKEN / GH_TOKEN  可选，用于提高 GitHub API 限额
 *   UPSTREAM_PROXY           可选，下载 FPK 时使用的 GitHub 代理前缀
 */

import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { mkdtemp, readFile, rm, stat, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const SOURCE_FILE = path.join(ROOT, 'fnpack.json');
const APP_KEY = 'fnmedia-ext';
const UPSTREAM_OWNER = 'yujian0815';
const UPSTREAM_REPO = 'fnmedia-ext';
const MAX_RELEASES = 100;
const VERSION_RE = /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;

const argv = process.argv.slice(2);
const dryRun = argv.includes('--dry-run');
const force = argv.includes('--force');
const slim = argv.includes('--slim');
const tagArg = optionValue('--tag');
const fpkArg = optionValue('--fpk');
const proxyArg = optionValue('--proxy');
const proxy = normalizeProxy(proxyArg || process.env.UPSTREAM_PROXY || '');

function optionValue(name) {
  const index = argv.indexOf(name);
  if (index === -1) return '';
  const value = argv[index + 1];
  if (!value || value.startsWith('--')) {
    throw new Error(`${name} 需要一个参数`);
  }
  return value;
}

function normalizeProxy(value) {
  if (!value) return '';
  if (!/^https?:\/\//i.test(value)) {
    throw new Error(`代理地址必须以 http(s):// 开头：${value}`);
  }
  return value.endsWith('/') ? value : `${value}/`;
}

function apiHeaders(accept = 'application/vnd.github+json') {
  const headers = {
    Accept: accept,
    'User-Agent': 'latte03-fnos-apps-sync',
  };
  const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN;
  if (token) headers.Authorization = `Bearer ${token}`;
  return headers;
}

async function fetchText(url, headers) {
  const response = await fetch(url, { headers, redirect: 'follow' });
  if (!response.ok) {
    throw new Error(`请求失败 HTTP ${response.status}: ${url}`);
  }
  return response.text();
}

async function fetchJson(url) {
  return JSON.parse(await fetchText(url, apiHeaders()));
}

async function loadRelease() {
  if (tagArg) {
    const url =
      `https://api.github.com/repos/${UPSTREAM_OWNER}/${UPSTREAM_REPO}/releases/tags/` +
      encodeURIComponent(tagArg);
    console.log(`读取上游 Release：${url}`);
    return fetchJson(url);
  }

  const url =
    `https://api.github.com/repos/${UPSTREAM_OWNER}/${UPSTREAM_REPO}/releases` +
    `?per_page=20`;
  console.log(`读取上游 Release 列表：${url}`);
  const releases = await fetchJson(url);
  if (!Array.isArray(releases)) {
    throw new Error('GitHub Releases API 返回的不是数组');
  }
  const release = releases.find((item) => !item.draft);
  if (!release) {
    throw new Error('上游没有可用的非草稿 Release');
  }
  return release;
}

async function sha256ForAsset(asset) {
  const digest = typeof asset.digest === 'string' ? asset.digest : '';
  const match = digest.match(/^sha256:([0-9a-f]{64})$/i);
  if (match) return match[1].toLowerCase();

  console.log(`  ${asset.name} 未提供 digest，下载后计算 sha256...`);
  const response = await fetch(asset.browser_download_url, {
    headers: {
      Accept: 'application/octet-stream',
      'User-Agent': 'latte03-fnos-apps-sync',
    },
    redirect: 'follow',
  });
  if (!response.ok) {
    throw new Error(`下载失败 HTTP ${response.status}: ${asset.browser_download_url}`);
  }
  const buffer = Buffer.from(await response.arrayBuffer());
  if (buffer.length !== asset.size) {
    throw new Error(
      `${asset.name} 大小不一致：API ${asset.size}，实际 ${buffer.length}`,
    );
  }
  return createHash('sha256').update(buffer).digest('hex');
}

async function downloadAsset(asset, destination) {
  const url = proxy ? `${proxy}${asset.browser_download_url}` : asset.browser_download_url;
  console.log(`下载 ${asset.name}...`);
  const response = await fetch(url, {
    headers: {
      Accept: 'application/octet-stream',
      'User-Agent': 'latte03-fnos-apps-sync',
    },
    redirect: 'follow',
  });
  if (!response.ok) {
    throw new Error(`下载失败 HTTP ${response.status}: ${url}`);
  }
  const buffer = Buffer.from(await response.arrayBuffer());
  if (buffer.length !== asset.size) {
    throw new Error(
      `${asset.name} 大小不一致：API ${asset.size}，实际 ${buffer.length}`,
    );
  }
  const expected = await sha256ForAsset(asset);
  const actual = createHash('sha256').update(buffer).digest('hex');
  if (actual !== expected) {
    throw new Error(`${asset.name} sha256 不一致：${actual} != ${expected}`);
  }
  await writeFile(destination, buffer);
  return destination;
}

function readTarEntry(fpkPath, entry) {
  return execFileSync('tar', ['-xOzf', path.basename(fpkPath), entry], {
    cwd: path.dirname(fpkPath),
    encoding: 'utf8',
    maxBuffer: 4 * 1024 * 1024,
  });
}

function parseManifest(text) {
  const result = {};
  for (const line of text.split(/\r?\n/)) {
    const match = line.match(/^\s*([A-Za-z0-9_]+)\s*=\s*(.*?)\s*$/);
    if (!match) continue;
    let value = match[2];
    if (value.startsWith('"') && value.endsWith('"')) {
      value = value.slice(1, -1).replace(/\\"/g, '"').replace(/\\\\/g, '\\');
    }
    result[match[1]] = value;
  }
  return result;
}

function canonical(value) {
  if (Array.isArray(value)) {
    return `[${value.map(canonical).join(',')}]`;
  }
  if (value && typeof value === 'object') {
    const body = Object.keys(value)
      .sort()
      .map((key) => `${JSON.stringify(key)}:${canonical(value[key])}`)
      .join(',');
    return `{${body}}`;
  }
  return JSON.stringify(value);
}

function semverParts(version) {
  const withoutBuild = String(version).split('+')[0];
  const dash = withoutBuild.indexOf('-');
  const core = dash === -1 ? withoutBuild : withoutBuild.slice(0, dash);
  const pre = dash === -1 ? '' : withoutBuild.slice(dash + 1);
  const nums = core.split('.').map((part) => Number.parseInt(part, 10) || 0);
  return { nums, pre };
}

function compareSemver(a, b) {
  const left = semverParts(a);
  const right = semverParts(b);
  for (let index = 0; index < 3; index += 1) {
    const delta = (left.nums[index] || 0) - (right.nums[index] || 0);
    if (delta !== 0) return delta;
  }
  if (left.pre === right.pre) return 0;
  if (!left.pre) return 1;
  if (!right.pre) return -1;
  return left.pre.localeCompare(right.pre);
}

function releaseTime(release) {
  const time = Date.parse(release.updated_at || '');
  return Number.isFinite(time) ? time : 0;
}

function sortReleases(releases) {
  const entries = Object.entries(releases).sort(([versionA, releaseA], [versionB, releaseB]) => {
    const timeDelta = releaseTime(releaseB) - releaseTime(releaseA);
    if (timeDelta !== 0) return timeDelta;
    return compareSemver(versionB, versionA);
  });
  return Object.fromEntries(entries);
}

async function main() {
  let tempDir = '';
  try {
    const release = await loadRelease();
    const version = String(release.tag_name || '').replace(/^v/i, '');
    if (!VERSION_RE.test(version)) {
      throw new Error(`无法识别的版本号：${release.tag_name}`);
    }

    const expectedName = slim
      ? `fnmedia-ext-${version}.fpk`
      : `fnmedia-ext-${version}-full.fpk`;
    const asset = release.assets.find((item) => item.name === expectedName);
    if (!asset) {
      throw new Error(
        `Release ${release.tag_name} 中找不到 ${expectedName}` +
          (slim ? '' : '；如要公开精简版请加 --slim'),
      );
    }
    console.log(`使用${slim ? '公开精简版' : '完整版'}：${asset.name}`);

    let fpkPath = fpkArg ? path.resolve(fpkArg) : '';
    if (fpkPath) {
      const info = await stat(fpkPath);
      if (!info.isFile()) {
        throw new Error(`--fpk 不是文件：${fpkPath}`);
      }
      if (info.size !== asset.size) {
        throw new Error(
          `--fpk 大小与 Release 不一致：${info.size} != ${asset.size}`,
        );
      }
      const expected = await sha256ForAsset(asset);
      const actual = createHash('sha256')
        .update(await readFile(fpkPath))
        .digest('hex');
      if (actual !== expected) {
        throw new Error(`--fpk sha256 与 Release 不一致：${actual} != ${expected}`);
      }
      console.log(`使用本地 FPK：${fpkPath}`);
    } else {
      tempDir = await mkdtemp(path.join(os.tmpdir(), 'fnmedia-ext-'));
      fpkPath = path.join(tempDir, path.basename(asset.name));
      await downloadAsset(asset, fpkPath);
    }

    const manifest = parseManifest(readTarEntry(fpkPath, 'manifest'));
    if (manifest.appname !== APP_KEY) {
      throw new Error(`manifest appname 为 ${manifest.appname}，预期 ${APP_KEY}`);
    }
    if (manifest.version !== version) {
      throw new Error(
        `manifest version 为 ${manifest.version}，Release 为 ${version}`,
      );
    }
    if (manifest.platform && manifest.platform !== 'all') {
      throw new Error(`manifest platform 为 ${manifest.platform}，预期 all`);
    }

    let privilege = null;
    try {
      privilege = JSON.parse(readTarEntry(fpkPath, 'config/privilege'));
    } catch {
      console.warn('  FPK 中没有可读取的 config/privilege，保留现有 run_as');
    }

    const source = JSON.parse(await readFile(SOURCE_FILE, 'utf8'));
    const app = source.apps?.[APP_KEY];
    if (!app) {
      throw new Error(`fnpack.json 中缺少 apps.${APP_KEY}`);
    }

    const before = canonical(source);

    const runAs = privilege?.defaults?.['run-as'] || app.run_as;
    if (runAs !== 'package' && runAs !== 'root') {
      throw new Error(`无法识别的 run-as：${runAs}`);
    }
    const servicePort = String(manifest.service_port ?? app.service_port ?? '');
    if (servicePort !== '' && !/^\d+$/.test(servicePort)) {
      throw new Error(`无法识别的 service_port：${servicePort}`);
    }

    app.display_name = manifest.display_name || app.display_name;
    app.desc = manifest.desc || app.desc;
    app.platform = ['all'];
    app.run_as = runAs;
    app.install_type = manifest.install_type || app.install_type || '';
    app.is_docker = false;
    app.service_port = servicePort;

    const nextRelease = {
      changelog: (release.body || '').trim() || `更新至 ${version}。`,
      updated_at:
        release.published_at || release.created_at || new Date().toISOString(),
    };
    if (manifest.os_min_version) {
      nextRelease.os_min_version = manifest.os_min_version;
    } else if (app.releases?.[version]?.os_min_version) {
      nextRelease.os_min_version = app.releases[version].os_min_version;
    }
    nextRelease.packages = {
      all: {
        download_url: asset.browser_download_url,
        sha256: await sha256ForAsset(asset),
        size: asset.size,
      },
    };

    const existing = app.releases?.[version];
    if (existing && canonical(existing) !== canonical(nextRelease)) {
      if (!force) {
        throw new Error(
          `版本 ${version} 已存在且内容不同；确认要覆盖请加 --force`,
        );
      }
      console.log(`覆盖已存在版本 ${version}`);
    }

    app.releases = app.releases || {};
    app.releases[version] = nextRelease;
    app.releases = sortReleases(app.releases);

    const versions = Object.keys(app.releases);
    if (versions.length > MAX_RELEASES) {
      for (const oldVersion of versions.slice(MAX_RELEASES)) {
        delete app.releases[oldVersion];
        console.log(`超出 ${MAX_RELEASES} 个版本上限，移除 ${oldVersion}`);
      }
    }

    if (canonical(source) === before) {
      console.log(`已是最新：${APP_KEY} ${version}`);
      return;
    }

    const output = `${JSON.stringify(source, null, 2)}\n`;
    if (dryRun) {
      process.stdout.write(output);
      console.log('\n[dry-run] 未写入文件');
      return;
    }

    await writeFile(SOURCE_FILE, output, 'utf8');
    console.log(
      `已写入 fnpack.json：${APP_KEY} ${version} (all ${asset.size} B)`,
    );
  } finally {
    if (tempDir) {
      await rm(tempDir, { recursive: true, force: true });
    }
  }
}

main().catch((error) => {
  console.error(`同步失败：${error.message}`);
  process.exit(1);
});
