#!/usr/bin/env python3
"""tools/make_icons.py — 生成 cloudflare-tunnel 的全部位图图标（无外部素材、无文字/emoji）。

设计：fnOS 友好的圆角方形，深蓝 → 靛蓝 → 青绿对角渐变底，白色「隧道拱门 + 箭头」意象。
与仓库根 fpk/icon.svg 同一套几何比例（512/1024 viewBox 等价）。

产物（相对 fpk/）：
    ui/images/icon-64.png    桌面/面板用（ui/config 的 images/icon-{0}.png）
    ui/images/icon-256.png   桌面/面板 2x
    ICON.PNG                 商店图标 256
    ICON_256.PNG             商店图标 256
    ui/images/icon-1024.png  源图留档（512 以上商店位图可用）

依赖：Pillow（纯 Python，无需 SVG 栅格化器）。
用法：python tools/make_icons.py
"""
from __future__ import annotations

import os
import sys
from pathlib import Path

try:
    from PIL import Image, ImageDraw
except ImportError:  # pragma: no cover
    sys.stderr.write("缺少 Pillow：python -m pip install pillow\n")
    raise SystemExit(1)

FPK_ROOT = Path(__file__).resolve().parents[1]

BASE = 1024          # 逻辑尺寸（= icon.svg 的 viewBox）
SS = 2               # 超采样倍数（2x 画完再缩，边缘干净）
W = BASE * SS

# 配色取自 prototype/styles.css（--accent / --teal / --ink）
STOPS = [
    (0.00, (0x0B, 0x1E, 0x49)),   # 深海军蓝
    (0.55, (0x1B, 0x46, 0xC9)),   # 靛蓝 #1b46c9
    (1.00, (0x0E, 0x93, 0x84)),   # 青绿 #0e9384
]
WHITE = (255, 255, 255, 255)


def gradient_color(t: float) -> tuple[int, int, int]:
    """分段线性插值。t=0 左下 → t=1 右上。"""
    t = max(0.0, min(1.0, t))
    for i in range(len(STOPS) - 1):
        t0, c0 = STOPS[i]
        t1, c1 = STOPS[i + 1]
        if t <= t1:
            k = 0.0 if t1 == t0 else (t - t0) / (t1 - t0)
            return tuple(round(c0[j] + (c1[j] - c0[j]) * k) for j in range(3))
    return STOPS[-1][1]


def make_gradient(size: int) -> Image.Image:
    """先在 160x160 上算渐变再双三次放大，避免逐像素跑 2048²。"""
    small_n = 160
    small = Image.new("RGB", (small_n, small_n))
    px = small.load()
    for y in range(small_n):
        for x in range(small_n):
            t = (x + (small_n - 1 - y)) / (2.0 * (small_n - 1))
            px[x, y] = gradient_color(t)
    return small.resize((size, size), Image.BICUBIC)


def circle(draw: ImageDraw.ImageDraw, x: float, y: float, r: float) -> None:
    draw.ellipse([x - r, y - r, x + r, y + r], fill=WHITE)


def build_icon() -> Image.Image:
    img = Image.new("RGBA", (W, W), (0, 0, 0, 0))

    # 1) 圆角方形底（半径 224/1024 ≈ iOS 超椭圆观感，fnOS 商店/桌面都吃这个比例）
    mask = Image.new("L", (W, W), 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, W - 1, W - 1], radius=round(W * 224 / 1024), fill=255)
    img.paste(make_gradient(W), (0, 0), mask)

    draw = ImageDraw.Draw(img)
    stroke = round(W * 87 / 1024)
    half_stroke = stroke / 2.0

    # 2) 拱门：两段立柱 + 上半圆
    cx = W * 0.48
    half = W * 0.23
    base_y = W * 0.74
    leg_top = W * 0.39

    draw.arc(
        [cx - half, leg_top - half, cx + half, leg_top + half],
        start=180, end=360, fill=WHITE, width=stroke,
    )
    for sx in (-1, 1):
        x = cx + sx * half
        draw.line([(x, leg_top), (x, base_y)], fill=WHITE, width=stroke)
        # 立柱底端做圆头（Pillow 的线没有线帽）
        circle(draw, x, base_y, half_stroke)

    # 3) 箭头：横穿门洞的箭头（箭杆 + 三角头），不碰拱门立柱
    s = W / 1024.0
    shaft_y = 589 * s
    draw.line([(345 * s, shaft_y), (566 * s, shaft_y)], fill=WHITE, width=stroke)
    circle(draw, 345 * s, shaft_y, half_stroke)
    draw.polygon(
        [(566 * s, 511 * s), (655 * s, 589 * s), (566 * s, 667 * s)],
        fill=WHITE,
    )

    return img.resize((BASE, BASE), Image.LANCZOS)


def save(img: Image.Image, rel: str, size: int) -> None:
    path = FPK_ROOT / rel
    path.parent.mkdir(parents=True, exist_ok=True)
    out = img if size == img.width else img.resize((size, size), Image.LANCZOS)
    out.save(path, format="PNG", optimize=True)
    print(f"  {rel:<28} {size}x{size}  {os.path.getsize(path)} B")


def main() -> int:
    # 输出全用 ASCII：Windows 控制台默认 GBK，中文会乱码，构建日志也更好 grep
    print(f"make_icons: writing PNGs into {FPK_ROOT}")
    icon = build_icon()
    save(icon, "ui/images/icon-1024.png", 1024)
    save(icon, "ui/images/icon-256.png", 256)
    save(icon, "ui/images/icon-64.png", 64)
    save(icon, "ICON.PNG", 256)
    save(icon, "ICON_256.PNG", 256)
    print("make_icons: done")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
