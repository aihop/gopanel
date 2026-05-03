from pathlib import Path
import sys

FIBER_IMPORT = '"github.com/gofiber/fiber/v2"'


def fix_file(path_str: str) -> None:
    path = Path(path_str)
    text = path.read_text()
    if "fiber.Ctx" not in text or FIBER_IMPORT in text:
        return
    marker = "import (\n"
    if marker not in text:
        raise SystemExit(f"no import block in {path}")
    text = text.replace(marker, marker + "\t" + FIBER_IMPORT + "\n", 1)
    path.write_text(text)


if __name__ == "__main__":
    for arg in sys.argv[1:]:
        fix_file(arg)
