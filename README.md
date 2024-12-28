# pixi

This repository contains the specification of a the Pixi file format. To start, think of Pixi like an opinionated cloud optimized GeoTIFF, but with explicit support for more than two dimensions and fewer built-in specifics for image-particular interpretation concerns.

## Design Considerations

1. More than images: Pixi is a format for tiled multidimensional raster data, which can be in dimensions higher than 2 or even 3. The interpretation of each dimension, even for dimensions 2 and 3, is defined on a per-file basis and cannot necessarily be assumed. Viewer applications may assume conventions for such files, but this is not specified by this document.

1. Queryable: it should be possible to transmit only a portion of a file (a 'tile') without needing to transmit the rest of the data. This is especially important for cloud optimized scenarios or files that are accessed by many different machines.

1. Robustness to transmission errors: it should be possible to detect datastream transmission errors reliably.

1. Portability: encoding, decoding, and transmission should be software and hardware platform independent.

1. Performance: any filtering and compression should be aimed at efficient decoding. Fast encoding is a less important goal than fast decoding. Decoding speed may be achieved at the expense of encoding speed.

1. Compression: files should be compressed effectively, consistent with the other design goals.

1. Interchangeability: any standard-conforming Pixi decoder shall be capable of reading all conforming Pixi datastreams.

1. Freedom from legal restrictions: no algorithms should be used that are not freely available.

## Terminology

## Concepts

### Fields

### Separation

### Dimensions and Tiling

### Layers

### Robustness and Errors

## Layout

## Compression

## Conformance

## Viewers

## Editors