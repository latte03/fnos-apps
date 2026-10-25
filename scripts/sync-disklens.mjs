#!/usr/bin/env node
/**
 * 跟随 JoySec123/DiskLens 的最新 Release，更新 fnpack.json。
 *
 * 历史 Release 的校验值也会自动核对：上游在原 tag 重传资产时刷新
 * size/sha256；上游删除的版本会从清单移除，保证 download_url 始终可用。
 *
 * 用法：
 *   node scripts/sync-disklens.mjs
 *   node scripts/sync-disklens.mjs --dry-run
 *   node scripts/sync-disklens.mjs --tag v0.3.3
 *   node scripts/sync-disklens.mjs --force
 *
 * 环境变量：
 *   GITHUB_TOKEN / GH_TOKEN  可选，用于提高 GitHub API 限额
 */

import { createHash } from 'node:crypto';
import { readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { refreshHistoricalPins, sameDownloadUrls } from './lib/upstream.mjs';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const SOURCE_FILE = path.join(ROOT, 'fnpack.json');
const APP_KEY = 'DiskLens';
const UPSTREAM_OWNER = 'JoySec123';
const UPSTREAM_REPO = 'DiskLens';
const MAX_RELEASES = 100;
const VERSION_RE = /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;

const argv = process.argv.slice(2);
const dryRun = argv.includes('--dry-run');
const force = argv.includes('--force');
const tagArg = optionValue('--tag');

function optionValue(name) {
  const index = argv.indexOf(name);
  if (index === -1) return '';
  const value = argv[index + 1];
  if (!value || value.startsWith('--')) {
    throw new Error(`${name} 需要一个参数`);
  }
  return value;
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

function parseManifest(text) {
  const result = {};
  for (const line of text.split(/\r?\n/)) {
    const match = line.match(/^\s*([A-Za-z0-9_]+)\s*=\s*"((?:[^"\\]|\\.)*)"\s*$/);
    if (!match) continue;
    result[match[1]] = match[2].replace(/\\"/g, '"').replace(/\\\\/g, '\\');
  }
  return result;
}

function pickAsset(release, expectedName, suffix) {
  const exact = release.assets.find((asset) => asset.name === expectedName);
  if (exact) return exact;

  const candidates = release.assets.filter((asset) =>
    asset.name.toLowerCase().endsWith(suffix),
  );
  if (candidates.length === 1) {
    console.warn(`  未找到 ${expectedName}，使用 ${candidates[0].name}`);
    return candidates[0];
  }
  throw new Error(`Release ${release.tag_name} 中找不到 ${expectedName}`);
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

async function fetchUpstreamManifest(tag) {
  const rawUrl =
    `https://raw.githubusercontent.com/${UPSTREAM_OWNER}/${UPSTREAM_REPO}/` +
    `${encodeURIComponent(tag)}/fnos/manifest`;
  console.log(`读取上游 manifest：${rawUrl}`);
  try {
    return await fetchText(rawUrl, {
      'User-Agent': 'latte03-fnos-apps-sync',
    });
  } catch (error) {
    console.warn(`raw 读取失败（${error.message}），改用 GitHub API`);
  }

  const apiUrl =
    `https://api.github.com/repos/${UPSTREAM_OWNER}/${UPSTREAM_REPO}/contents/` +
    `fnos/manifest?ref=${encodeURIComponent(tag)}`;
  const data = await fetchJson(apiUrl);
  if (!data || typeof data.content !== 'string') {
    throw new Error('GitHub API 未返回 manifest 内容');
  }
  return Buffer.from(data.content, data.encoding || 'base64').toString('utf8');
}

async function loadRelease() {
  if (tagArg) {
    const url =
      `https://api.github.com/repos/${UPSTREAM_OWNER}/${UPSTREAM_REPO}/releases/tags/` +
      encodeURIComponent(tagArg);
    console.log(`读取上游 Release：${url}`);
    return fetchJson(url);
  }

  // /releases/latest 会跳过预发布版本；DiskLens 的测试版正是预发布，
  // 因此从列表里取最新的非草稿 Release。
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

async function main() {
  const release = await loadRelease();

  const version = String(release.tag_name || '').replace(/^v/i, '');
  if (!VERSION_RE.test(version)) {
    throw new Error(`无法识别的版本号：${release.tag_name}`);
  }

  const x86Asset = pickAsset(
    release,
    `DiskLens-${version}-amd64.fpk`,
    '-amd64.fpk',
  );
  const armAsset = pickAsset(
    release,
    `DiskLens-${version}-arm64.fpk`,
    '-arm64.fpk',
  );

  console.log(`版本 ${version}，核对安装包摘要...`);
  const x86 = {
    download_url: x86Asset.browser_download_url,
    sha256: await sha256ForAsset(x86Asset),
    size: x86Asset.size,
  };
  const arm = {
    download_url: armAsset.browser_download_url,
    sha256: await sha256ForAsset(armAsset),
    size: armAsset.size,
  };

  const manifest = parseManifest(
    await fetchUpstreamManifest(release.tag_name),
  );

  if (manifest.appname !== APP_KEY) {
    throw new Error(`manifest appname 为 ${manifest.appname}，预期 ${APP_KEY}`);
  }
  if (manifest.version !== version) {
    throw new Error(`manifest version 为 ${manifest.version}，Release 为 ${version}`);
  }

  const source = JSON.parse(await readFile(SOURCE_FILE, 'utf8'));
  const app = source.apps?.[APP_KEY];
  if (!app) {
    throw new Error(`fnpack.json 中缺少 apps.${APP_KEY}`);
  }

  const before = canonical(source);

  const nextRelease = {
    changelog: manifest.changelog || release.name || `更新至 ${version}。`,
    updated_at: release.published_at || release.created_at || new Date().toISOString(),
    packages: { x86, arm },
  };

  const existing = app.releases?.[version];
  if (existing && canonical(existing) !== canonical(nextRelease)) {
    if (!sameDownloadUrls(existing, nextRelease) && !force) {
      throw new Error(
        `版本 ${version} 已存在且下载地址变化；确认要覆盖请加 --force`,
      );
    }
    console.log(
      sameDownloadUrls(existing, nextRelease)
        ? `上游更新了 ${version}（同一下载地址），刷新校验值与元数据`
        : `覆盖已存在版本 ${version}`,
    );
  }

  if (manifest.display_name) app.display_name = manifest.display_name;
  if (manifest.desc) app.desc = manifest.desc;

  app.releases = app.releases || {};
  app.releases[version] = nextRelease;
  await refreshHistoricalPins(app, version, {
    owner: UPSTREAM_OWNER,
    repo: UPSTREAM_REPO,
    fetchJson,
    sha256ForAsset,
  });
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
    console.log(`\n[dry-run] 未写入文件`);
    return;
  }

  await writeFile(SOURCE_FILE, output, 'utf8');
  console.log(
    `已写入 fnpack.json：${APP_KEY} ${version} ` +
      `(x86 ${x86.size} B, arm ${arm.size} B)`,
  );
}

main().catch((error) => {
  console.error(`同步失败：${error.message}`);
  process.exit(1);
});
