/* Copyright (C) 2017 Matthias S. Benkmann
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this file (originally named buttons.go) and associated documentation files 
 * (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is furnished
 * to do so, subject to the following conditions:
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 * 
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE. 
 */

// Manages graphics and sound assets.
package ass

import "os"
import "io/ioutil"
import "path"
import "strings"

// Superinterface of all assets (graphics, sound,...).
type Asset interface{
  // Unmarshal's the JSON metadata of the asset into target.
  Meta(target interface{}) error
}

type pile struct {
  asset Asset
  sub map[string]*pile
}

// All Assets are stored in this pile.
var assets = &pile{sub:map[string]*pile{}}

// Superinterface of all graphics assets.
type ImageAsset interface{
  Asset
  // Renders the image with the given width*height into an RGBA array.
  Render(width,height int32) ([]uint32,error)
}

// A rectangular part of an SVG image.
type SVGAsset struct {
  // XML source code up to the location where viewBox and/or width/height
  // attributes for the svg element need to be inserted. Never includes
  // viewBox/width/height. Always ends in whitespace, so no additional
  // whitespace needs to be inserted before viewBox.
  Head []byte
  
  // viewBox="..." attribute to be inserted between Head and Body.
  ViewBox []byte
  
  // XML source code following the location where viewBox attribute for
  // svg element needs to be inserted.
  Body []byte
  
  // Metadata in JSON format. Always includes "x","y","width","height","centerx"
  // and "centery".
  MetaJSON []byte
}

// If pth is a directory, recursively scans it and subdirectories and collects
// assets found. If pth refers to an asset file, only that one is added.
func Add(pth string) error {
  d, err := os.Open(pth)
  if err != nil { return err }
  defer d.Close()
  
  fi, err := d.Stat()
  if err != nil { return err }
  
  if fi.IsDir() {
    fis, err := d.Readdir(-1)
    if err != nil { return err }
    
    // Don't hold file open unnecessarily.
    d.Close()
    
    for _, fi := range fis {
      err = Add(path.Join(pth,fi.Name()))
      if err != nil { return err }
    }
  } else {
    pth = strings.ToLower(path.Clean(pth))
    if path.Ext(pth) == ".svg" {
      data, err := ioutil.ReadAll(d)
      if err != nil { return err }
      addSVG(pth[0:len(pth)-len(".svg")], data)
    }
  }
  return nil
}

// Returns a list (unsorted) of the full paths of all assets with the given path_prefix.
// If prefix does not end in "/" it is nevertheless assumed. IOW, a path_prefix
// cannot be a partial name.
// The returned paths DO NOT start with "/" (and path_prefix may but need not
// start with a "/", either).
// If no assets are found, the return value is nil.
func List(path_prefix string) []string {
  pth := strings.ToLower(path.Clean(path_prefix))
  if pth == "/" { pth = "" }
  pths := strings.Split(pth,"/")
  if pths[0] == "" { pths = pths[1:] } // if pth starts with "/"
  pil := assets
  for _, p := range pths {
    pil = pil.sub[p]
    if pil == nil { return nil }
  }
  
  res := make([]string,0,len(assets.sub))
  list(&res, pil, pths)
  return res
}

func list(res *[]string, p *pile, prefix []string) {
  if p.asset != nil {
    *res = append(*res, strings.Join(prefix,"/"))
  }
  for k,s := range p.sub {
    list(res, s, append(prefix, k))
  }
}

// Unmarshal's the JSON metadata of the asset with the given asset_path into
// target.
func Meta(asset_path string, target interface{}) error {
  pil := find(asset_path)
  if pil == nil { return os.ErrNotExist }
  return pil.asset.Meta(target)
}

// Renders the image asset with the given asset_path into an RGBA array
// with the given width*height.
func Image(asset_path string, width, height int32) ([]uint32, error) {
  pil := find(asset_path)
  if pil == nil { return nil, os.ErrNotExist }
  var imass ImageAsset
  imass, ok := pil.asset.(ImageAsset)
  if !ok { return nil, ErrAssetType }
  return imass.Render(width,height)
}

// Returns the pile for path pth if it exists AND has an asset. Otherwise returns nil.
func find(pth string) *pile {
  pth = strings.ToLower(path.Clean(pth))
  if pth == "/" { pth = "" }
  pths := strings.Split(pth,"/")
  if pths[0] == "" { pths = pths[1:] } // if pth starts with "/"
  pil := assets
  for _, p := range pths {
    pil = pil.sub[p]
    if pil == nil { return nil }
  }
  if pil.asset == nil { return nil }
  return pil
}




