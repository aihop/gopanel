#!/usr/bin/env python3
"""
解码 SQL 导出文件中被序列化为 []byte 整数切片的字段。

问题：Go 的 gorm 将 string/json 字段以 []byte 形式存入数据库，
      导出时显示为 "[119 120 51 52 ...]" 而不是可读文本。

用法：
    python3 script/decode_sql_bytes.py < input.sql > output.sql
    # 或直接修改文件
    python3 script/decode_sql_bytes.py -i dump.sql
"""

import re
import sys
import argparse

# 匹配 '[num num num ...]' 的字节数组格式
# 括号内是空格分隔的 0-255 整数
BYTE_ARR_RE = re.compile(r"'\[(\d+(?: \d+)*)\]'")


def decode_byte_array(match: re.Match) -> str:
    """
    将匹配到的字节数组替换为可读字符串。
    如果解码后不可读（含控制字符），则保留原始格式。
    """
    raw = match.group(1)
    nums = [int(x) for x in raw.split()]
    try:
        decoded = bytes(nums).decode('utf-8')
        # 检查是否包含不可见的控制字符（排除常见的空白符）
        # 允许: 中文、ASCII 可打印字符、Tab/换行/回车
        if all(ord(c) >= 32 or ord(c) in (9, 10, 13) for c in decoded):
            return f"'{decoded}'"
        # 有不可见控制字符，保留原样
        return match.group(0)
    except (UnicodeDecodeError, ValueError):
        return match.group(0)


def decode_sql(input_text: str) -> str:
    """替换输入文本中所有字节数组为字符串。"""
    return BYTE_ARR_RE.sub(decode_byte_array, input_text)


def main():
    parser = argparse.ArgumentParser(
        description="解码 SQL 中 Go []byte 序列化的整数切片为可读字符串"
    )
    parser.add_argument(
        "-i", "--inplace",
        help="直接修改指定文件（原地替换）",
        dest="inplace_file"
    )
    args = parser.parse_args()

    if args.inplace_file:
        with open(args.inplace_file, 'r', encoding='utf-8') as f:
            content = f.read()
        decoded = decode_sql(content)
        with open(args.inplace_file, 'w', encoding='utf-8') as f:
            f.write(decoded)
        print(f"已解码并写入: {args.inplace_file}")
    else:
        # 管道模式
        content = sys.stdin.read()
        sys.stdout.write(decode_sql(content))


if __name__ == "__main__":
    main()
