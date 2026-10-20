#!/usr/bin/env node
/**
 * 按 FnDepot V2 源规范校验 fnpack.json。
 *
 * 用法：
 *   node scripts/validate.mjs
 *   node scripts/validate.mjs --remote   # 额外下载安装包核对 sha256 / size
 */

import { createHash } from 'node:crypto';
import { open, readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const SOURCE_FILE = path.join(ROOT, 'fnpack.json');

const LIMITS = {
  mainBytes: 2 * 1024 * 1024,
  apps: 5000,
  versions: 100,
  iconBytes: 500 * 1024,
  previewBytes: 2 * 1024 * 1024,
  readmeBytes: 1024 * 1024,
};

const CATEGORIES = new Set([
  '影音娱乐',
  '系统工具',
  '编程开发',
  'AI赋能',
  '生活服务',
  '智能智控',
  '教育学习',
  '游戏地带',
  '硬件驱动',
]);
const PLATFORMS = new Set(['all', 'x86', 'arm']);
const PACKAGE_KEYS = new Set(['all', 'x86', 'arm']);
const APP_KEY_RE = /^[A-Za-z0-9][A-Za-z0-9._-]*$/;
const VERSION_RE = /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;
const SHA256_RE = /^[0-9a-f]{64}$/i;

const remote = process.argv.includes('--remote');
const errors = [];
const warnings = [];

function error(message) {
  errors.push(message);
}

function warn(message) {
  warnings.push(message);
}

function isPlainObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function requireString(value, label, { allowEmpty = false } = {}) {
  if (typeof value !== 'string') {
    error(`${label} 必须是字符串`);
    return false;
  }
  if (!allowEmpty && value.trim() === '') {
    error(`${label} 不能为空`);
    return false;
  }
  return true;
}

function checkUrl(value, label, { required = true } = {}) {
  if (value === undefined || value === null || value === '') {
    if (required) error(`${label} 不能为空`);
    return;
  }
  if (typeof value !== 'string') {
    error(`${label} 必须是字符串`);
    return;
  }
  if (/^https?:\/\//i.test(value) || value.startsWith('./') || value.startsWith('../')) {
    return;
  }
  error(`${label} 必须是 http(s) 绝对地址或以 ./ ../ 开头的相对地址：${value}`);
}

function localFilePath(url) {
  if (typeof url !== 'string') return null;
  if (/^https?:\/\//i.test(url)) return null;
  if (!url.startsWith('./') && !url.startsWith('../')) return null;

  let decoded;
  try {
    decoded = decodeURIComponent(url.split('#')[0].split('?')[0]);
  } catch {
    error(`URL 编码无效：${url}`);
    return null;
  }

  const full = path.resolve(ROOT, decoded);
  if (full !== ROOT && !full.startsWith(ROOT + path.sep)) {
    error(`相对地址指向仓库外：${url}`);
    return null;
  }
  return full;
}

async function checkLocalAsset(url, label, maxBytes) {
  const file = localFilePath(url);
  if (!file) return;

  let info;
  try {
    info = await stat(file);
  } catch {
    error(`${label} 文件不存在：${url}`);
    return;
  }
  if (!info.isFile()) {
    error(`${label} 不是文件：${url}`);
    return;
  }
  if (info.size > maxBytes) {
    error(`${label} 超过大小限制 ${maxBytes} B：${info.size} B`);
  }

  const extension = path.extname(file).toLowerCase();
  if (label === 'icon_url') {
    if (!['.png', '.webp'].includes(extension)) {
      error(`icon_url 必须是 PNG 或 WebP：${url}`);
      return;
    }
    const handle = await open(file, 'r');
    const head = Buffer.alloc(12);
    await handle.read(head, 0, 12, 0);
    await handle.close();
    const isPng = head.subarray(0, 8).equals(Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]));
    const isWebp =
      head.subarray(0, 4).toString('ascii') === 'RIFF' &&
      head.subarray(8, 12).toString('ascii') === 'WEBP';
    if (!isPng && !isWebp) {
      error(`icon_url 内容不是有效的 PNG/WebP：${url}`);
    }
  }
}

function checkPlatform(value, label) {
  const list = Array.isArray(value) ? value : [value];
  if (list.length === 0) {
    error(`${label} 不能为空`);
    return [];
  }
  if (new Set(list).size !== list.length) {
    error(`${label} 不能包含重复值`);
  }
  for (const item of list) {
    if (typeof item !== 'string' || !PLATFORMS.has(item)) {
      error(`${label} 只能是 all、x86、arm：${JSON.stringify(item)}`);
    }
  }
  return list.filter((item) => PLATFORMS.has(item));
}

function checkCategories(value) {
  if (!Array.isArray(value)) {
    error('categories 必须是数组');
    return [];
  }
  if (value.length < 1 || value.length > 2) {
    error('categories 必须有 1 到 2 个分类');
  }
  if (new Set(value).size !== value.length) {
    error('categories 不能包含重复分类');
  }
  for (const item of value) {
    if (typeof item !== 'string' || !CATEGORIES.has(item)) {
      error(`categories 含未知分类：${JSON.stringify(item)}`);
    }
  }
  return value;
}

async function verifyRemotePackage(downloadUrl, sha256, size, label) {
  console.log(`下载校验 ${label}：${downloadUrl}`);
  const response = await fetch(downloadUrl, {
    headers: {
      Accept: 'application/octet-stream',
      'User-Agent': 'latte03-fnos-apps-validator',
    },
    redirect: 'follow',
  });
  if (!response.ok) {
    error(`${label} 下载失败 HTTP ${response.status}`);
    return;
  }
  const buffer = Buffer.from(await response.arrayBuffer());
  if (buffer.length !== size) {
    error(`${label} 大小不一致：清单 ${size} B，实际 ${buffer.length} B`);
  }
  const actual = createHash('sha256').update(buffer).digest('hex');
  if (actual !== sha256.toLowerCase()) {
    error(`${label} sha256 不一致：清单 ${sha256}，实际 ${actual}`);
  }
}

async function main() {
  let raw;
  try {
    raw = await readFile(SOURCE_FILE);
  } catch {
    error('缺少 fnpack.json');
    return;
  }
  if (raw.length > LIMITS.mainBytes) {
    error(`fnpack.json 超过 ${LIMITS.mainBytes} B`);
  }

  const text = raw.toString('utf8');
  if (text.includes('\uFFFD')) {
    error('fnpack.json 不是有效的 UTF-8 文本');
  }

  let source;
  try {
    source = JSON.parse(text);
  } catch (parseError) {
    error(`JSON 解析失败：${parseError.message}`);
    return;
  }

  if (!isPlainObject(source)) {
    error('顶层必须是对象');
    return;
  }
  if (source.schema_version !== '2') {
    error(`schema_version 必须是字符串 "2"，当前为 ${JSON.stringify(source.schema_version)}`);
  }
  if (!isPlainObject(source.source_info)) {
    error('缺少 source_info 对象');
  } else {
    requireString(source.source_info.name, 'source_info.name');
    requireString(source.source_info.author, 'source_info.author');
    if (source.source_info.author?.trim().toLowerCase() === 'fndepot') {
      error('source_info.author 不能冒充 fndepot');
    }
    if (source.source_info.homepage !== undefined) {
      checkUrl(source.source_info.homepage, 'source_info.homepage', { required: false });
    }
    if (source.source_info.description !== undefined) {
      requireString(source.source_info.description, 'source_info.description', { allowEmpty: true });
    }
  }

  if (!isPlainObject(source.apps)) {
    error('缺少 apps 对象');
    return;
  }

  const appEntries = Object.entries(source.apps);
  if (appEntries.length === 0) {
    error('apps 不能为空');
  }
  if (appEntries.length > LIMITS.apps) {
    error(`apps 数量超过 ${LIMITS.apps}`);
  }

  let releaseCount = 0;

  for (const [appKey, app] of appEntries) {
    const label = `apps.${appKey}`;
    if (!APP_KEY_RE.test(appKey)) {
      error(`${label} 键不合法（只能字母、数字、点、下划线、短横线，且以字母或数字开头）`);
    }
    if (!isPlainObject(app)) {
      error(`${label} 必须是对象`);
      continue;
    }

    requireString(app.display_name, `${label}.display_name`);
    requireString(app.desc, `${label}.desc`);

    if (app.platform === undefined) {
      error(`${label}.platform 缺失`);
    }
    const platforms = checkPlatform(app.platform, `${label}.platform`);

    if (app.categories === undefined) {
      error(`${label}.categories 缺失`);
    } else {
      checkCategories(app.categories);
    }

    if (app.icon_url === undefined) {
      error(`${label}.icon_url 缺失`);
    } else {
      checkUrl(app.icon_url, `${label}.icon_url`);
      await checkLocalAsset(app.icon_url, 'icon_url', LIMITS.iconBytes);
    }

    if (app.preview_urls !== undefined) {
      if (!Array.isArray(app.preview_urls)) {
        error(`${label}.preview_urls 必须是数组`);
      } else {
        if (app.preview_urls.length > 8) {
          warn(`${label}.preview_urls 超过 8 张，客户端只读取前 8 张`);
        }
        for (const [index, preview] of app.preview_urls.entries()) {
          checkUrl(preview, `${label}.preview_urls[${index}]`);
          await checkLocalAsset(preview, `preview_urls[${index}]`, LIMITS.previewBytes);
        }
      }
    }

    if (app.readme_url !== undefined) {
      checkUrl(app.readme_url, `${label}.readme_url`, { required: false });
      await checkLocalAsset(app.readme_url, 'readme_url', LIMITS.readmeBytes);
    }

    for (const field of ['bug_report_url', 'maintainer_url', 'distributor_url', 'details_url']) {
      if (app[field] !== undefined) {
        checkUrl(app[field], `${label}.${field}`, { required: false });
      }
    }
    for (const field of ['maintainer', 'distributor']) {
      if (app[field] !== undefined) {
        requireString(app[field], `${label}.${field}`, { allowEmpty: true });
        if (app[field]?.trim().toLowerCase() === 'fndepot') {
          error(`${label}.${field} 不能冒充 fndepot`);
        }
      }
    }

    if (app.run_as !== 'package' && app.run_as !== 'root') {
      error(`${label}.run_as 只能是 package 或 root`);
    }
    if (app.install_type !== '' && app.install_type !== 'root') {
      error(`${label}.install_type 只能是空字符串或 root`);
    }
    if (typeof app.is_docker !== 'boolean') {
      error(`${label}.is_docker 必须是布尔值`);
    }
    if (app.service_port !== undefined && app.service_port !== '') {
      if (!/^\d+$/.test(String(app.service_port))) {
        error(`${label}.service_port 必须是数字或空字符串`);
      }
    }

    if (!isPlainObject(app.releases)) {
      error(`${label}.releases 必须是对象`);
      continue;
    }

    const releaseEntries = Object.entries(app.releases);
    if (releaseEntries.length === 0) {
      error(`${label}.releases 不能为空`);
    }
    if (releaseEntries.length > LIMITS.versions) {
      error(`${label}.releases 超过 ${LIMITS.versions} 个版本`);
    }

    for (const [version, release] of releaseEntries) {
      releaseCount += 1;
      const releaseLabel = `${label}.releases.${version}`;
      if (version === 'latest' || !VERSION_RE.test(version)) {
        error(`${releaseLabel} 版本号不合法`);
      }
      if (!isPlainObject(release)) {
        error(`${releaseLabel} 必须是对象`);
        continue;
      }
      if (release.updated_at !== undefined && Number.isNaN(Date.parse(release.updated_at))) {
        warn(`${releaseLabel}.updated_at 不是可解析的时间：${release.updated_at}`);
      }
      if (release.changelog !== undefined) {
        requireString(release.changelog, `${releaseLabel}.changelog`, { allowEmpty: true });
      }

      if (!isPlainObject(release.packages)) {
        error(`${releaseLabel}.packages 必须是对象`);
        continue;
      }
      const packageEntries = Object.entries(release.packages);
      if (packageEntries.length === 0) {
        error(`${releaseLabel}.packages 不能为空`);
        continue;
      }

      for (const packageKey of Object.keys(release.packages)) {
        if (!PACKAGE_KEYS.has(packageKey)) {
          error(`${releaseLabel}.packages 含非法架构键：${packageKey}`);
        }
      }

      const available = new Set(Object.keys(release.packages));
      for (const platform of platforms) {
        if (platform === 'all') continue;
        if (!available.has(platform) && !available.has('all')) {
          error(`${releaseLabel} 声明支持 ${platform}，但缺少 packages.${platform} 或 packages.all`);
        }
      }

      for (const [packageKey, pkg] of packageEntries) {
        const packageLabel = `${releaseLabel}.packages.${packageKey}`;
        if (!PACKAGE_KEYS.has(packageKey)) continue;
        if (!isPlainObject(pkg)) {
          error(`${packageLabel} 必须是对象`);
          continue;
        }
        checkUrl(pkg.download_url, `${packageLabel}.download_url`);
        if (typeof pkg.sha256 !== 'string' || !SHA256_RE.test(pkg.sha256)) {
          error(`${packageLabel}.sha256 必须是 64 位十六进制字符串`);
        }
        if (!Number.isInteger(pkg.size) || pkg.size < 0) {
          error(`${packageLabel}.size 必须是非负整数（字节）`);
        }

        if (
          remote &&
          typeof pkg.download_url === 'string' &&
          /^https?:\/\//i.test(pkg.download_url) &&
          typeof pkg.sha256 === 'string' &&
          SHA256_RE.test(pkg.sha256) &&
          Number.isInteger(pkg.size)
        ) {
          await verifyRemotePackage(
            pkg.download_url,
            pkg.sha256,
            pkg.size,
            `${appKey} ${version} ${packageKey}`,
          );
        }
      }
    }
  }

  if (warnings.length > 0) {
    console.log('警告：');
    for (const message of warnings) console.log(`  - ${message}`);
  }
  if (errors.length > 0) {
    console.error('校验失败：');
    for (const message of errors) console.error(`  - ${message}`);
    process.exitCode = 1;
    return;
  }
  console.log(
    `校验通过：${appEntries.length} 个应用，${releaseCount} 个版本` +
      (remote ? '（已下载核对安装包）' : ''),
  );
}

main().catch((error) => {
  console.error(`校验异常：${error.message}`);
  process.exit(1);
});
