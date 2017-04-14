# assman
Management library (go lang) for audio and video assets for use with SDL

## Overview
You call the Add() function to add individual files or directories (which
will be scanned recursively). All assets found will be added to the database
identified by an asset path. The asset path is derived from the path of the
file the asset is found in in the following manner:

- The base path is the one that was passed to the Add() function, after
  normalization, e.g. Add("./foo/bar/../bar") will result in a base path of
  "foo/bar".
- The base path is split at '/'.
- Trailing digits are removed from each part of the base path, e.g.
  "foo123/bar456" becomes "foo/bar". This allows you to have multiple
  directories/files whose assets will be placed at the same position in the
  database tree. E.g. "flowers1.svg" and "flowers2.svg" could both contain
  flower images that would all be found as flowers/* in the asset database.
- All letters are converted to lower-case.
- Depending on the file type an individual file may contain multiple assets
  (possibly arranged in a hierarchy). In that case, their hierarchy and
  labels will be handled like directory components, i.e. they will also be
  converted to lower-case and have trailing digits removed.

## SVG files
- In order to be recognized, SVG files must have the ".svg" extension.
- Each .svg file produces at least 1 image asset whose name is the file name
  without ".svg" extension and with trailing digits (after removing the
  ".svg" extension) removed. The area of the SVG that forms the image is
  determined by the viewBox attribute of the top-level <svg> element.

In addition to the SVG file's master asset, it may contain multiple
sub-assets that extract various portitions of the SVG. These portions may be
outside of the main viewBox. Sub-assets can be arranged in a hierarchy, e.g.
a "car" image may have a sub-asset "wheel" whose rectangle is completely
inside the rectangle of the car. If the SVG file was called "vehicles.svg"
that would result in the sub-asset paths "vehicles/car" and
"vehicles/car/wheel".

In order to mark sub-assets in an SVG file, do the following in Inkscape
(steps may differ for other SVG editors):

- Create a layer called "METADATA" (all caps!).
- Inside that layer, create rectangles that cover the areas of your
  sub-assets. Note that transformations and other shenanigans are not
  supported for these rectangles. Use only plain rectangles created with the
  rectangle tool.
- Fill and stroke don't matter. Note that the area that counts for the asset
  is the area of the rectangle without stroke, so it is recommended that you
  use rectangles with only fill to see exactly what is covered.
- Tip: Make the METADATA layer partially transparent to see through your
  rectangles.
- Tip: By clicking the "eye" next to the name "METADATA" in the layers
  dialog you can make the METADATA layer and all its rectangles disappear,
  so that they don't interfere with your editing of the actual asset
  images.
- Transparency and visibility setting of the METADATA layer do not matter to
  assman, so you don't have to set them to a specific state when saving the
  file.
- In the Object Properties of your asset rectangles, set the "id" to the
  name of the asset. For consistency it is recommended that you set the
  "label" to a matching name. Don't forget to click the "Set" button for
  your changes to take effect.
- Remember: Trailing digits in asset names are removed when forming their
  asset paths.
- When a rectangle is fully contained inside a larger rectangle, it will be
  considered to be a sub-asset. This allows you to form an asset hierarchy.

When the Meta() function is used with the path of an image asset from an SVG
file, assman will always provide the following metadata:

- "x", "y": If the asset is a top-level asset, i.e. its rectangle is not
  fully contained in another rectangle (the master image's viewBox does NOT
  count), these coordinates are both 0. If the asset's rectangle is
  contained in another rectangle, these coordinates are the relative
  coordinates within the enclosing rectangle. Note that rectangles may be
  nested in multiple levels. The parent rectangle is always the one closest
  in size.
- "width", "height": Obvious.
- "centerx","centery": By default these are width/2 and height/2. However if
  you move Inkscape's rotation center (click on the rectangle when in select
  mode until the rotation arrows appear at the corners, then drag the little
  cross in the middle of the rectangle where you want it to be), that will
  override the default.

The description in the rectangle's Object Properties may be used to store
further metadata attributes. The format of the description field is as
follows:
- You may use JSON syntax for the whole or parts of the description field.
- {...} braces surrounding the whole description (as would be required for proper
  JSON) are optional.
- Commas (which separate individual JSON fields) at the end of lines may be
  omitted.
- You can use '=' instead of ':'.
- The first key of each line does not need to be enclosed in "..." quotes.
- A string that does not start with a digit or the strings "true", "false"
  or "null" does not need to be enclosed in "..." quotes. It will be
  assumed to extend to the end of the line.
- Everything starting with a '#' up to the end of the line is ignored,
  except if the '#' occurs within a string.

The following is a valid example of a description block:
```
a: "string value\n2nd line. Embedded \"quotes\"."
"b"= 42
c= true
  # comment
d: Hi, this works without "..."
e: { "f":"bar", "g": 99
     h: ["bla"
     "fasel",
     "dusel"
     ]
   }
```
