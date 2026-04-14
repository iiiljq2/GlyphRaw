# GlyphRaw

**GlyphRaw** is an automated command-line tool for generating personalized handwriting fonts from handwritten character images using the [FontDiffuser](https://github.com/yeungchenwa/FontDiffuser) deep learning model. Simply provide your handwriting samples, and GlyphRaw will generate a TTF or OTF font file.

## Features

- **Automated Font Generation**: Convert handwritten character images to professional fonts
- **Docker Integration**: Seamless Docker containerization for consistent environments
- **GPU Acceleration**: Full support for NVIDIA GPU acceleration via CUDA
- **Flexible Output**: Generate TTF or OTF font formats
- **Auto Setup**: Automatic download and initialization of models
- **Easy Configuration**: Simple command-line interface with sensible defaults

## Requirements

### System Requirements

- **Operating System**: Linux, macOS, or Windows
- **Go**: Version 1.21 or higher
- **Docker**: Version 20.0 or higher
- **GPU** (recommended): NVIDIA GPU with CUDA 11.7 support (minimum 4GB VRAM)

## Installation

### From Source

```bash
git clone https://github.com/iiiljq/glyphraw.git
cd glyphraw
go build -o glyphraw ./
```

On Windows:
```bash
go build -o glyphraw.exe ./
```

## Quick Start

```bash
./glyphraw
```

Follow the interactive prompts to generate your font.

## Usage

### Supported Formats

JPG, JPEG, PNG, BMP, TIFF

### Single File
```bash
./glyphraw
# When prompted: /path/to/my_handwriting.jpg
```

### Multiple Files (Directory)
```bash
./glyphraw
# When prompted: /path/to/my_handwriting_samples/
```

### Output

Generated fonts are saved in the `article_output/` directory.

## Project Structure

```
glyphraw/
├── main.go
├── internal/
│   ├── logger/          # Logging
│   ├── config/          # Configuration and models
│   ├── docker/          # Docker execution
│   ├── font/            # Font generation
│   ├── setup/           # Model downloads
│   └── cli/             # User interaction
├── pkg/
│   ├── util/            # Utilities
│   └── download/        # File downloads
├── scripts/
│   └── pack_font.py     # Font packing script
└── Dockerfile
```

## Troubleshooting

### Docker Not Running
Install Docker Desktop from https://www.docker.com/ and start it.

### Out of Disk Space
Models require ~2-3GB. Delete `checkpoints/` and `assets/` directories and restart.

### Low GPU Memory
Close other GPU-consuming applications. The model requires at least 4GB+ VRAM for stable performance.

### Missing Characters in Generated Font
Check `article_output/[stylename]/` for generated PNG files and review logs.

## Development

### Adding New Features

1. **New Model**: Add configuration to `internal/config/models.go`
2. **New Font Format**: Extend `internal/font/packer.go`
3. **CLI Enhancements**: Modify `internal/cli/interactive.go`

## Performance

**With GPU acceleration**:
- Single character: 1-3 seconds
- Full font (100+ characters): 5-10 minutes

## Known Limitations

1. Primarily supports Chinese characters
2. Significantly slower without GPU
3. Initial download ~2-3GB
4. Other scripts require model fine-tuning

## Related Projects

- [FontDiffuser](https://github.com/yeungchenwa/FontDiffuser) - Original deep learning model
- [FontForge](https://github.com/fontforge/fontforge) - Font generation engine

## Support
If you encounter any issues or have any suggestions for improvement, please feel free to reach out for assistance through the following channels:

**Discord**: iiiljq (username: hunter)

---

**Maintainer**: iiiljq
