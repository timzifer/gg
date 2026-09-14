# Stencil alpha comparison

These examples reproduce the darkening of translucent GPU stencil fills in gg v0.52.5. The images were rendered on Windows amd64 with Go 1.25.3. Both modules use wgpu v0.32.1, as in the earlier GPU repros on this branch.

- `repro-alpha`: gg-only drawing example with SDF rectangles, convex fills, concave fills, and multiple subpaths. Columns vary alpha from 1 to 0.125. It prints interior GPU/CPU pixels at alpha 89/255.
- `repro-alpha-violin`: the deterministic violin example from figure's gallery, with its default 35% fill opacity.

Each program renders the GPU image, closes the accelerator and unregisters the coverage filler, then renders the CPU reference. A failed GPU flush stops the program rather than silently producing a reference under a GPU label.

## Reproduce

For the original images, run each module outside a Go workspace:

```sh
cd repro-alpha
GOWORK=off go run . -out ../images/alpha-before
cd ../repro-alpha-violin
GOWORK=off go run . -out ../images/violin-alpha-before
```

For the fixed images, check out `timzifer/gg` branch `fix/gpu-stencil-alpha`. Use a local Go workspace to select that checkout without changing either example's go.mod:

```sh
cd repro-alpha
go work init . /path/to/gg-fix
go run . -out ../images/alpha-after
cd ../repro-alpha-violin
go work init . /path/to/gg-fix
go run . -out ../images/violin-alpha-after
```

On PowerShell, use `$env:GOWORK='off'` for the original run and remove that environment override for the workspace run. The workspace files are local configuration and are not part of this evidence commit.

## Measured interior colors

At alpha 89/255, over white:

| Shape | GPU before | GPU after | CPU |
|---|---|---|---|
| Blue rectangle | (166, 206, 228) | (166, 206, 228) | (166, 205, 228) |
| Orange convex fill | (240, 199, 166) | (240, 199, 166) | (240, 198, 166) |
| Blue concave fill | (166, 180, 188) | (166, 206, 228) | (166, 205, 228) |
| Orange disjoint subpaths | (192, 177, 166) | (240, 199, 166) | (240, 198, 166) |

The one-byte CPU/GPU difference is rounding. The two CPU reference PNGs for each example are byte-identical before and after the fix. Geometry and input colors are unchanged.
