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

### Layers

### Fields

### Separation

### Dimensions and Tiling

### Robustness and Errors

## Layout

This section details the byte-level layout of a Pixi file.

### Pixi Header

Every Pixi file should start with four bytes: "PIXI". This is followed by the version number, written as a number in UTF-8 string in four bytes. The version number should be prefixed with leading zeros if the printed number string is not long enough to fill two bytes.

Following this magic sequence and version number is the offset size indicator. This is a single byte, indicating the number of bytes that will make up offset values later in this file, used to point to different byte indices within the file. Currently, the only supported values of the offset size indicator are 4 and 8 (for 32-bit and 64-bit requirements respectively).

Then the endianness indicator follows, another single byte. This indicates the endianness of all multibyte values that follow in the data stream. The two supported options are little endian at 0x00 and big endian with 0xff.

Following this is the first layer offset, which will be an integer composed of the number of bytes specified by the offset size indicator. This will be the byte offset in the file, with index 0 equal to the start of the file, at which the first layer's first byte can be found.

Following this offset is the tagging offset. This will be the offset in the file at which the tagging section can start being read.

### Layer Header

### Tagging Section

### Field Header

## Compression

## Conformance

## Viewers

## Editors