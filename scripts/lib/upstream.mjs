/**
 * GitHub 上游 Release 共享工具：供各 sync-*.mjs 与 validate.mjs 复用。
 *
 * - githubAssetInfo(url)   解析 github.com/{owner}/{repo}/releases/download/{tag}/{file}
 * - assetDigest(asset)     读取 Release 资产的 sha256 digest（无则 null）
 * - sameDownloadUrls(a, b) 比较两个版本对象的下载地址是否一致
 * - refreshHistoricalPins(app, currentVersion, opts) 历史 pin 自愈
 */

const GITHUB_ASSET_RE =
  /^https:\/\/github\.com\/([^/]+)\/([^/]+)\/releases\/download\/([^/]+)\/([^?#]+)/i;

export function githubAssetInfo(downloadUrl) {
  const match = String(downloadUrl).match(GITHUB_ASSET_RE);
  if (!match) return null;
  return {
    owner: match[1],
    repo: match[2],
    tag: match[3],
    filename: decodeURIComponent(match[4]),
  };
}

export function assetDigest(asset) {
  const digest = typeof asset?.digest === 'string' ? asset.digest : '';
  const match = digest.match(/^sha256:([0-9a-f]{64})$/i);
  return match ? match[1].toLowerCase() : null;
}

export function sameDownloadUrls(releaseA, releaseB) {
  const keysA = Object.keys(releaseA?.packages || {});
  const keysB = Object.keys(releaseB?.packages || {});
  if (keysA.length !== keysB.length) return false;
  return keysA.every(
    (key) =>
      releaseA.packages[key]?.download_url === releaseB.packages[key]?.download_url,
  );
}

/**
 * 上游偶尔会在同一个 tag 下原地重传资产（fnmedia-ext 0.2.5 就被重传过），
 * 历史 pin 过期会让 validate --remote 失败、整个流水线卡住。
 * 逐个核对历史版本（跳过 currentVersion）：资产变化时刷新 size / sha256
 * （优先用 GitHub API 的 digest 字段，避免重复下载）；上游已删除 tag 或
 * 资产的版本从清单移除（多架构版本任一资产缺失即整体移除，否则过不了
 * 平台覆盖校验）。tag 与资产名均取自 download_url 本身，不做 v 前缀假设。
 *
 * opts：
 *   owner / repo            上游仓库
 *   fetchJson(url)          脚本现有的 GitHub API GET（带 token）；
 *                           404 时抛错且 message 含 "HTTP 404"
 *   sha256ForAsset(asset)   API 无 digest 时下载计算 sha256 的兜底函数
 */
export async function refreshHistoricalPins(app, currentVersion, opts) {
  const { owner, repo, fetchJson, sha256ForAsset } = opts;
  const prefix = `https://github.com/${owner}/${repo}/releases/download/`;
  const releaseCache = new Map();

  const loadRelease = async (tag) => {
    if (releaseCache.has(tag)) return releaseCache.get(tag);
    const url =
      `https://api.github.com/repos/${owner}/${repo}/releases/tags/` +
      encodeURIComponent(tag);
    let release;
    try {
      release = await fetchJson(url);
    } catch (err) {
      if (/HTTP 404/.test(err.message)) {
        releaseCache.set(tag, undefined);
        return undefined;
      }
      throw err;
    }
    releaseCache.set(tag, release);
    return release;
  };

  for (const oldVersion of Object.keys(app.releases || {})) {
    if (oldVersion === currentVersion) continue;
    const packages = app.releases[oldVersion]?.packages || {};
    const entries = Object.entries(packages).filter(
      ([, pkg]) =>
        typeof pkg?.download_url === 'string' && pkg.download_url.startsWith(prefix),
    );
    if (entries.length === 0) continue;

    for (const [packageKey, pkg] of entries) {
      const info = githubAssetInfo(pkg.download_url);
      if (!info) continue;

      const upstreamRelease = await loadRelease(info.tag);
      if (!upstreamRelease) {
        delete app.releases[oldVersion];
        console.log(`上游已删除 ${info.tag}，移除版本 ${oldVersion}`);
        break;
      }

      const asset = (upstreamRelease.assets || []).find(
        (item) => item.name === info.filename,
      );
      if (!asset) {
        delete app.releases[oldVersion];
        console.log(
          `上游 ${info.tag} 已不存在资产 ${info.filename}，移除版本 ${oldVersion}`,
        );
        break;
      }

      const digest = assetDigest(asset);
      const shaSame = digest
        ? digest === String(pkg.sha256 || '').toLowerCase()
        : null;
      // 大小一致且 sha256 一致（API 无 digest 时以大小一致为准）则无需刷新
      if (asset.size === pkg.size && shaSame !== false) continue;

      pkg.size = asset.size;
      pkg.sha256 = digest ?? (await sha256ForAsset(asset));
      console.log(
        `上游重传了 ${info.filename}，刷新 ${oldVersion} ${packageKey} 的 size/sha256`,
      );
    }
  }
}
