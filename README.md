# imgstat

Image color analysis from the command line.

Reads JPEG, PNG, GIF, BMP, TIFF, and WebP. Format is detected from file
content, not extension. With no arguments, reads standard input.

## Usage

```
imgstat [-n count] [-k clusters] [file ...]
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-n` | 20 | top N colors in the histogram |
| `-k` | 6 | k-means palette clusters (0 to disable) |

## Output

```
# photo.jpg jpeg 800x600 480000px 12345 colors
#    hex   r   g   b       n      %
  2b3a4c  43  58  76   12345   2.6%
  ...
# mean   128 128 126  luma 126.9±42.1  sat 0.67  entropy 13.29
# stddev  74  74  74  cast R+12 G-5 B-7  colorful 45.2  dynrange 0.87
# sharp 142.3  edges 12.3%
# palette k=6
#    hex   r   g   b      %
  2b3a4c  43  58  76  45.2%
```

`#` lines are comments. Hex values paste directly into HTML/CSS.

### Metrics

| Field | Description |
|-------|-------------|
| `luma ±` | BT.601 perceived brightness, mean ± std dev |
| `sat` | mean HSL saturation (0 = grey, 1 = fully saturated) |
| `entropy` | Shannon entropy in bits; 0 = solid color, ~13 = noise |
| `cast R G B` | gray-world deviation per channel; positive = warm shift |
| `colorful` | Hasler–Süsstrunk colorfulness index |
| `dynrange` | (max − min luma) / 255 |
| `sharp` | variance of Laplacian; near-zero = blurry |
| `edges %` | fraction of pixels above Sobel gradient threshold |
| `palette` | k-means dominant colors, merging near-identical hues |

## Examples

```sh
imgstat photo.jpg
imgstat -n 5 *.png
cat image.webp | imgstat
imgstat -k 8 a.jpg b.png c.gif
```

## Build

```sh
go build .
```

Requires Go 1.22+.

## Exit status

- `0` — all files decoded successfully
- `1` — one or more files could not be opened or decoded
- `2` — wrong parameters passed in the command line

## License

MIT. Copyright 2024 Adam Koszek.
