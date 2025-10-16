
import os
from PIL import Image
import argparse


def pad_pngs_in_dir(target_dir: str, save_subdir_name: str = "Texture") -> int:
    """遍历 target_dir（非递归），把目录下的每个 PNG 补齐到偶数宽高并保存到同一目录下的 `save_subdir_name` 子目录。

    返回处理的文件数量。
    """
    target_dir = os.path.abspath(target_dir)
    if not os.path.isdir(target_dir):
        raise NotADirectoryError(target_dir)

    saved = 0
    # 列出目录中的 png 文件并排序，避免不稳定顺序
    files = sorted([f for f in os.listdir(target_dir) if f.lower().endswith('.png')])
    if not files:
        return 0

    out_dir = os.path.join(target_dir, save_subdir_name)
    os.makedirs(out_dir, exist_ok=True)

    for fname in files:
        src_path = os.path.join(target_dir, fname)
        try:
            img = Image.open(src_path)
        except Exception:
            # 跳过无法打开的文件
            continue

        w, h = img.size
        new_w = w + (w % 2)
        new_h = h + (h % 2)

        if new_w == w and new_h == h:
            # 已经是偶数，不需要修改，直接复制到输出目录
            img.save(os.path.join(out_dir, fname))
            saved += 1
            continue

        # 选择填充色：RGBA 用透明 (0,0,0,0)，其他模式用 0
        mode = img.mode
        if 'A' in mode:
            fill = (0,) * len(img.getbands())
        else:
            fill = 0

        new_img = Image.new(mode, (new_w, new_h), fill)
        new_img.paste(img, (0, 0))
        new_img.save(os.path.join(out_dir, fname))
        saved += 1

    return saved


def main():
    parser = argparse.ArgumentParser(description='Pad PNG files in a directory to even width/height and save to a Texture subfolder.')
    parser.add_argument('dir', nargs='?', default='.', help='目标目录 (默认: 当前目录)')
    args = parser.parse_args()

    count = pad_pngs_in_dir(args.dir)
    print(f"处理完成：{count} 个 PNG 文件已保存到子目录 'Texture'。")


if __name__ == '__main__':
    main()