import os
import sys
import fontforge

def generate_ttf(input_dir, output_ttf):
    font = fontforge.font()
    font.encoding = "UnicodeFull"

    font_name = os.path.basename(os.path.normpath(input_dir))
    font.fontname = font_name
    font.fullname = font_name
    font.familyname = font_name

    print(f"[Task] Packing font from: {input_dir}")

    success_count = 0

    for char_name in os.listdir(input_dir):
        char_path = os.path.join(input_dir, char_name)

        if not os.path.isdir(char_path):
            continue

        if len(char_name) != 1:
            continue

        png_path = os.path.join(char_path, f"{char_name}_single.png")
        if not os.path.exists(png_path):
            continue

        try:
            unicode_val = ord(char_name)
            glyph = font.createChar(unicode_val, char_name)

            glyph.importOutlines(png_path)
            glyph.autoTrace()
            glyph.addExtrema()
            glyph.simplify()

            success_count += 1
            if success_count % 100 == 0:
                print(f"  - Progress: {success_count} characters processed...")

        except Exception as e:
            print(f"  [Warning] Failed to process character '{char_name}': {e}")
            continue

    if success_count == 0:
        print("[Error] No valid character images found. Font was not generated.")
        return

    # Export to TTF
    font.generate(output_ttf)
    print(f"[Success] Font generated with {success_count} characters.")
    print(f"[Success] Saved to: {output_ttf}")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python pack_font.py <input_dir> <output_ttf_path>")
        sys.exit(1)

    generate_ttf(sys.argv[1], sys.argv[2])