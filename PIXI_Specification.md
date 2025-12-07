# PIXI File Format Specification

## 1. Introduction and Scope

### 1.1 Motivation and Goals

The PIXI file format is designed for the efficient storage and transmission of large multidimensional, multichannel gridded datasets. Traditional image formats are optimized for two-dimensional, generally colorized, visual data, while scientific and analytical applications increasingly require support for higher-dimensional data with multiple channels or fields per data point.

PIXI provides several key advantages over existing formats:

- **Cloud-optimized access**: Enables efficient partial data retrieval without downloading entire files, crucial for distributed computing and remote analysis
- **Multidimensional support**: Native support for datasets with arbitrary dimensionality (not limited to 2D or 3D)
- **Flexible data organization**: Supports both separated (field-by-field) and contiguous (interleaved) data layouts
- **Robust compression**: Multiple compression algorithms optimized for different data characteristics
- **Data integrity**: Built-in checksums ensure reliable data transmission and storage
- **Extensible metadata**: Flexible tagging system for domain-specific metadata

The format prioritizes fast decoding performance, making it suitable for real-time analysis and visualization applications while maintaining effective compression ratios.

### 1.2 Scope and Audience

This specification defines the complete binary layout and semantics of PIXI files, targeting:

- **Format implementers**: Developers creating PIXI encoders, decoders, and processing libraries
- **Application developers**: Engineers building analysis tools, visualization software, and data processing pipelines
- **Data scientists and researchers**: Users working with large multidimensional datasets who need efficient storage and access patterns
- **System architects**: Professionals designing data infrastructure for scientific computing and analytics

The specification covers file structure, data layout, compression methods, metadata handling, and conformance requirements. While reference implementations exist in Go, this specification is language-agnostic and implementation-independent.

### 1.3 Conventions and Terminology

This document uses the following key terms:

**Data Layer**: A logical grouping of related data within a PIXI file, typically representing the same dataset at different resolutions or zoom levels. Each layer contains its own dimensions, fields, and tile organization. Multiple layers enable efficient multi-resolution access patterns.

**Dimension**: An axis along which gridded data is organized (e.g., X, Y, Z coordinates, time, spectral bands). Each dimension has a total size and tile size that determines how data is chunked for storage. Dimensions are ordered such that the first dimension varies most rapidly in memory layout.

**Channel**: A distinct data component or measurement at each grid point, such as red/green/blue color values, temperature/pressure readings, or different sensor measurements. Each channel has a specific data type (integer, floating-point) and bit depth.
*Note: In the current Go implementation, channels are referred to as "fields" and will be renamed to "channels" in future versions.*

**Tile/Chunk**: A contiguous block of data that represents a rectangular region across all dimensions. Tiles enable efficient partial data access and are the unit of compression and I/O operations. Tile boundaries need not align with dimension boundaries, allowing flexible data organization.

**Header**: The metadata section at the beginning of a PIXI file containing format version, byte order, offset specifications, and pointers to the first layer and tag sections. The header enables proper interpretation of the remaining file contents.

**Tags**: Key-value string pairs providing extensible metadata about the file, its contents, provenance, or processing history. Tags are organized in linked sections to support efficient appending without rewriting the entire file.

**Separation**: A data layout option determining whether channels are stored together (contiguous/interleaved) or separately. Separated storage groups all values for a single channel together, while contiguous storage interleaves channel values at each grid point.

**Compression**: Data encoding methods applied to tiles to reduce storage requirements. PIXI supports multiple algorithms including FLATE, LZW variants, and run-length encoding, allowing optimization for different data characteristics.

**Field**: Legacy term for channel in the current implementation. Will be replaced with "channel" in future versions to align with common terminology in multidimensional data formats.

**Offset**: A file position indicator specifying the byte location of data structures within the file. Offsets enable random access to layers, tiles, and metadata sections.

## 2. Core Data Model

The PIXI format defines a hierarchical logical structure for organizing multidimensional data, independent of the physical file layout. Understanding this logical model is essential for working with PIXI data effectively.

### 2.1 Hierarchy Overview

PIXI data follows a four-level hierarchy:

```
PIXI File
├── Data Layer(s)
│   ├── Channels
│   ├── Dimensions
│   └── Samples
```

**PIXI File**: The top-level container holding one or more data layers plus optional metadata tags.

**Data Layer**: A logical grouping of related multidimensional data, typically representing different resolutions or zoom levels of the same dataset. Each layer is independent and self-contained.

**Channels**: The data components measured or recorded at each point in the multidimensional grid (e.g., RGB color values, temperature and humidity readings, or multiple sensor measurements).

**Dimensions**: The axes defining the multidimensional coordinate system (e.g., X/Y spatial coordinates, Z depth, time, spectral bands).

**Samples**: Individual data points containing one value for each channel at a specific coordinate location in the multidimensional space.

### 2.2 Dimension Ordering and Coordinate Systems

Dimensions in PIXI are strictly ordered, with this ordering affecting both memory layout and coordinate interpretation:

1. **Memory Layout**: The first dimension varies most rapidly in memory, followed by the second dimension, and so on. This row-major ordering ensures spatial locality for efficient access patterns.

2. **Coordinate Systems**: Multiple coordinate representations are supported:
   - **Sample Coordinates**: Direct multidimensional indices `[d₀, d₁, ..., dₙ]` where each element indexes into the corresponding dimension
   - **Linear Indices**: Single integer indices that can be converted to/from sample coordinates
   - **Tile Coordinates**: Two-level addressing `{tile: [t₀, t₁, ..., tₙ], inTile: [s₀, s₁, ..., sₙ]}` specifying both tile location and position within tile

3. **Coordinate Conversion**: The format provides deterministic mapping between coordinate systems:
   ```
   Sample Coordinate [2, 1, 0] in 3D space (4×3×2)
   ↓ (with tile sizes 2×2×1)
   Tile Coordinate {tile: [1, 0, 0], inTile: [0, 1, 0]}
   ↓
   Linear Index: 6 (= 2×3×0 + 1×3 + 0×1 + 2)
   ```

### 2.3 Supported Data Types

PIXI supports a comprehensive set of numeric data types for channel values:

**Integer Types**:
- **int8/uint8**: 8-bit signed/unsigned integers (1 byte each)
- **int16/uint16**: 16-bit signed/unsigned integers (2 bytes each)  
- **int32/uint32**: 32-bit signed/unsigned integers (4 bytes each)
- **int64/uint64**: 64-bit signed/unsigned integers (8 bytes each)

**Floating-Point Types**:
- **float32**: 32-bit IEEE 754 single-precision floating-point (4 bytes)
- **float64**: 64-bit IEEE 754 double-precision floating-point (8 bytes)

All multi-byte values are stored according to the endianness specified in the file header, ensuring consistent interpretation across different hardware architectures.

### 2.4 Channel Storage Patterns

PIXI supports two distinct storage patterns for organizing channel data within each sample:

#### 2.4.1 Contiguous (Interleaved) Storage
- **Pattern**: `[c₀, c₁, ..., cₙ, c₀, c₁, ..., cₙ, ...]`
- **Description**: All channel values for a single sample are stored together before moving to the next sample
- **Advantages**: Efficient when accessing all channels of individual samples simultaneously
- **Use Cases**: RGB image data, multi-sensor readings at single locations

#### 2.4.2 Separated (Planar) Storage  
- **Pattern**: `[c₀, c₀, ..., c₀, c₁, c₁, ..., c₁, ..., cₙ, cₙ, ..., cₙ]`
- **Description**: All values for a single channel across all samples are stored together
- **Advantages**: Efficient when accessing single channels across many samples; often better compression ratios
- **Use Cases**: Scientific datasets where individual measurements are analyzed separately

### 2.5 Samples and Data Access

**Sample Definition**: A sample represents the complete set of channel values at a specific coordinate location. Each sample contains exactly one value per channel, ordered according to the channel definition sequence.

**Sample Size**: The total byte size of a sample equals the sum of all channel data type sizes. For example, a sample with three float32 channels requires 12 bytes (3 × 4 bytes).

**Access Patterns**: The logical model supports multiple access patterns:
- **Sequential Access**: Iterate through all samples in dimension order
- **Random Access**: Direct access to specific samples via coordinate conversion
- **Channel-Specific Access**: Access single channels across sample ranges (efficient with separated storage)
- **Tile-Based Access**: Access rectangular regions via tile coordinate system

This logical model provides the foundation for efficient data organization while remaining independent of the specific physical file layout and compression strategies.

## 3. Physical File Structure

The PIXI format defines a specific binary layout for storing the logical data model on disk. This section describes the byte-level encoding of all file components.

### 3.1 Overview and File Components

A PIXI file consists of the following components in order:

```
┌─────────────────┐
│   File Header   │  Fixed-size header with format metadata
├─────────────────┤
│   Data Layer 1  │  Variable-size layer header + tile data
├─────────────────┤
│   Data Layer 2  │  Additional layers (optional)
├─────────────────┤
│       ...       │
├─────────────────┤
│  Tag Section 1  │  Variable-size metadata sections
├─────────────────┤
│  Tag Section 2  │  Additional tag sections (optional)  
├─────────────────┤
│       ...       │
└─────────────────┘
```

**Key Principles**:
- **Random Access**: Offsets enable direct access to any component without parsing the entire file
- **Extensibility**: New layers and tag sections can be appended without rewriting existing data
- **Self-Describing**: All size and offset information is embedded within the file structure
- **Platform Independence**: Explicit endianness and size specifications ensure cross-platform compatibility

### 3.2 File Header

The file header is always located at byte offset 0 and has a fixed structure:

```
Offset | Size | Type      | Description
-------|------|-----------|------------------------------------------
0      | 4    | ASCII     | Magic bytes: "pixi" (0x70, 0x69, 0x78, 0x69)
4      | 2    | ASCII     | Version number as 2-digit string (e.g., "01")
6      | 1    | uint8     | Offset size indicator (4 or 8)
7      | 1    | uint8     | Endianness (0x00=little, 0xFF=big)
8      | N    | offsetN   | First layer offset (N = offset size)
8+N    | N    | offsetN   | First tags offset (N = offset size)
```

**Total Header Size**: 8 + 2×(offset size) bytes

**Field Details**:

- **Magic Bytes**: Always the ASCII string "pixi" to identify the file format
- **Version**: Two-digit decimal version number with leading zero padding (current version: "01")
- **Offset Size**: Number of bytes used for all file offsets (4 for files <4GB, 8 for larger files)
- **Endianness**: Byte order for all multi-byte integers:
  - `0x00`: Little-endian (least significant byte first)
  - `0xFF`: Big-endian (most significant byte first)
- **First Layer Offset**: File position where the first data layer begins (0 if no layers)
- **First Tags Offset**: File position where the first tag section begins (0 if no tags)

### 3.3 Tag Sections

Tag sections store key-value string metadata and form a linked list structure:

```
Offset | Size | Type      | Description
-------|------|-----------|------------------------------------------
0      | 4    | uint32    | Number of tag pairs (N)
4      | var  | string    | Tag pair 1: key (friendly string)
       | var  | string    | Tag pair 1: value (friendly string)
       | var  | string    | Tag pair 2: key (friendly string)
       | var  | string    | Tag pair 2: value (friendly string)
       | ...  | ...       | ... (N total pairs)
       | M    | offsetN   | Next tag section offset (0 if last)
```

**Friendly String Format**:
```
Offset | Size | Type      | Description
-------|------|-----------|------------------------------------------
0      | 2    | uint16    | String length in bytes (L)
2      | L    | UTF-8     | String data
```

**Tag Section Features**:
- **Variable Size**: Sections grow based on number and length of tag pairs
- **UTF-8 Encoding**: All strings use UTF-8 encoding for international character support
- **Linked Structure**: Next offset allows chaining multiple tag sections
- **Appendable**: New tag sections can be added without modifying existing ones

### 3.4 Data Layers

Data layers contain the actual gridded data organized as tiles. Each layer consists of a header followed by tile data:

#### 3.4.1 Layer Header Structure

```
Offset | Size | Type      | Description
-------|------|-----------|------------------------------------------
0      | 4    | uint32    | Configuration flags
4      | 4    | uint32    | Compression algorithm
8      | var  | string    | Layer name (friendly string)
       | 4    | uint32    | Number of dimensions (D)
       | var  | dimension | Dimension 1 definition
       | var  | dimension | Dimension 2 definition
       | ...  | ...       | ... (D total dimensions)
       | 4    | uint32    | Number of channels (C)
       | var  | channel   | Channel 1 definition  
       | var  | channel   | Channel 2 definition
       | ...  | ...       | ... (C total channels)
       | T×N  | offsetN   | Tile byte counts (T tiles, N-byte offsets)
       | T×N  | offsetN   | Tile file offsets (T tiles, N-byte offsets)
       | N    | offsetN   | Next layer offset (0 if last layer)
```

**Configuration Flags** (uint32):
- Bit 0: Separation flag (0=contiguous, 1=separated channel storage)
- Bits 1-31: Reserved (must be 0)

**Compression Algorithms** (uint32):
- 0: None (uncompressed)
- 1: FLATE (RFC 1951)
- 2: LZW LSB (Lempel-Ziv-Welch, least significant bit first)
- 3: LZW MSB (Lempel-Ziv-Welch, most significant bit first)
- 4: RLE8 (Run-length encoding, up to 255 repetitions)

#### 3.4.2 Dimension Definition

```
Offset | Size | Type      | Description
-------|------|-----------|------------------------------------------
0      | var  | string    | Dimension name (friendly string)
       | N    | offsetN   | Total dimension size
       | N    | offsetN   | Tile size for this dimension
```

#### 3.4.3 Channel Definition

```
Offset | Size | Type      | Description  
-------|------|-----------|------------------------------------------
0      | var  | string    | Channel name (friendly string)
       | 4    | uint32    | Channel data type
```

**Channel Data Types** (uint32):
- 1: int8 (1 byte signed integer)
- 2: uint8 (1 byte unsigned integer)  
- 3: int16 (2 byte signed integer)
- 4: uint16 (2 byte unsigned integer)
- 5: int32 (4 byte signed integer)
- 6: uint32 (4 byte unsigned integer)
- 7: int64 (8 byte signed integer)
- 8: uint64 (8 byte unsigned integer)
- 9: float32 (4 byte IEEE 754 single precision)
- 10: float64 (8 byte IEEE 754 double precision)

#### 3.4.4 Tile Layout

Each tile contains compressed sample data followed by a checksum:

```
┌─────────────────────┐
│  Compressed Data    │  Variable size, format depends on compression
├─────────────────────┤
│  CRC32 Checksum     │  4 bytes, checksum of uncompressed data
└─────────────────────┘
```

**Tile Data Organization**:

For **Contiguous (Interleaved)** storage:
```
Sample 0: [Ch0, Ch1, ..., ChN]
Sample 1: [Ch0, Ch1, ..., ChN]
...
Sample M: [Ch0, Ch1, ..., ChN]
```

For **Separated (Planar)** storage:
```
Channel 0: [S0, S1, ..., SM]  (All samples for channel 0)
Channel 1: [S0, S1, ..., SM]  (All samples for channel 1)
...
Channel N: [S0, S1, ..., SM]  (All samples for channel N)
```

**Tile Indexing**:
- **Contiguous**: Number of disk tiles = Number of logical tiles
- **Separated**: Number of disk tiles = Number of logical tiles × Number of channels

**Data Integrity**: Each tile includes a CRC32 checksum of the uncompressed data (computed using the IEEE polynomial) to detect corruption during storage or transmission.