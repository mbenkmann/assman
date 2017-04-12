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

/*
#cgo pkg-config: cairo librsvg-2.0
#include <cairo.h>
#include <librsvg/rsvg.h>
#include <stdlib.h>
*/
import "C"


import (
         "strings"
         "fmt"
         "html"
         "sort"
         "math"
         "errors"
         "unsafe"
         "strconv"
         "encoding/json"
         
         "github.com/veandco/go-sdl2/sdl"
         "github.com/mbenkmann/golib/util"
)

// Adds an SVG image stored in data to the database with path pth.
// Errors are appended to ShitLog.
func addSVG(pth string, data []byte) {
  defer func() {
    if recover() != nil {
      ShitLog = append(ShitLog, fmt.Sprintf("%v: Not a well-formed XML file",pth))
      return
    }
  }()

  id := strings.Split(pth,"/")
  if id[0] == "" { id = id[1:] } // in case pth starts with "/"
  for i := range id {
    // remove trailing digits
    id[i] = strings.TrimRight(id[i], "0123456789")
    if id[i] == "" {
      ShitLog = append(ShitLog, fmt.Sprintf("%v: All path components must contain at least 1 non-digit character",pth))
      return
    }
  }
  
  // We copy from data[in] to data[out]. Because we remove whitespace and certain parts of
  // the image out <= in.
  in := 0
  out := 0
  
  // Nesting level. Incremented on <tag> and decremented on </tag>.
  level := 0
  
  // If kill_level >= 0 and level is decremented to become kill_level, then
  // out is set to kill_out. kill_out is a copy of out at the point where the
  // "<" of the start tag of the element to be killed was written.
  kill_level := -1
  kill_out := 0
  
  // Most recent start tag name, without namespace prefix.
  tagname := ""
  
  // data[start] points to the "<" of the most recent start tag (in the output buffer)
  start := 0
  
  // true while between "<" and ">" of a tag.
  in_tag := false
  
  // Most recent attribute name, without namespace prefix.
  attrname := ""
  
  // true while within the <g> element with id/label "METADATA".
  in_metadata := false
  
  // Collects the attributes of the most recent start tag.
  attributes := map[string]string{}
  
  // Each <rect> element within the <g> with id/label "METADATA" has its attributes appended here.
  // In addition to the element attributes, if the <rect> has a <desc> child, that element's
  // content is stored under the name "description" in the respective map.
  metadata := []map[string]string{}
  
  // Set to the index of the ">" of the outermost <svg> element.
  svgelement := 0
  
  // The attributes of the outermost <svg> element.
  toplevelmeta := map[string]string{}
  
  for {
    c := data[in]
    in++
    if c == '<' {
      d := data[in]
      if d == '?' { // copy <?xml verbatim
        for c != '>' {
          data[out] = c
          out++
          c = data[in]
          in++
        }
      } else if d == '!' { // skip <!--
        for c != '>' {
          c = data[in]
          in++
        }
        continue
      } else if d == '/' { // </end>
        o := out
        etn := out+2
        for c != '>' {
          data[out] = c
          out++
          c = data[in]
          in++
          if c == ':' { // skip namespace prefix
            etn = out+1
          }
        }
        endtagname := string(data[etn:out])
        if endtagname == "desc" {
          desc := o
          for data[desc-1] != '>' { desc-- }
          attributes["description"] = html.UnescapeString(string(data[desc:o]))
        } else if in_metadata && endtagname == "rect" {
          metadata = append(metadata,attributes)
        }
        level--
        if level == 0 { // end of document
          data[out] = '>'
          out++
          break
        }
        if level == kill_level {
          out = kill_out
          kill_level = -1
          in_metadata = false
          continue
        }
      } else { // <start ...> or <empty .../>
        in_tag = true
        start = out
        tagnamestart := start
        for c > ' ' && c != '/' && c != '>' {
          data[out] = c
          out++
          c = data[in]
          in++
          if c == ':' { // do not include namespace prefix
            tagnamestart = out
          }
        }
        tagname = string(data[tagnamestart+1:out])
        if tagname == "rect" {
          attributes = map[string]string{}
        }
        level++
        if c == '>' || c == '/' { // if we have just <foo> or <foo/ we need to process the character after "foo"
          in--                    // so take a step back
          continue
        }
      }
    } else if !in_metadata && (c == ' ' || c == '\t' || c == '\n' || c == '\r') {
      d := data[in]
      for d == ' ' || d == '\t' || d == '\n' || d == '\r' { // compress sequences of whitespace
        in++
        d = data[in]
      }
      c = ' '
    } else if c == '"' || c == '\'' { // quoted string
      data[out] = c
      out++
      attr := out
      d := data[in]
      in++
      for d != c {
        data[out] = d
        out++
        d = data[in]
        in++
      }
      if in_tag  { 
        attrval := string(data[attr:out])
        
        // remove viewBox, width and height from top-level <svg> element
        if level == 1 && (attrname == "viewBox" || attrname == "width" || attrname == "height") {
          toplevelmeta[attrname] = attrval
          for data[out-1] != c { out-- }
          for data[out-1] > ' ' { out-- }
          continue
        }
        
        if tagname == "rect" { attributes[attrname] = attrval }
        if tagname == "g" && (attrname == "id" || attrname == "label") && // a group with an all-uppercase label or id is eliminated from output
           attrval == strings.ToUpper(attrval) && kill_level < 0 {
          kill_level = level-1
          kill_out = start
          in_metadata = (attrval == "METADATA")
        }
      }
    } else if in_tag {
      if c == '/' {  // ..../>
        if in_metadata && tagname == "rect" {
          metadata = append(metadata,attributes)
        }
        data[out] = c
        out++
        c = data[in] // this is supposed to be '>'
        in++
        level--
        if level == kill_level {
          out = kill_out
          kill_level = -1
          in_metadata = false
          continue
        }
      }
      
      if c == '>' {
        in_tag = false
        if level == 1 && svgelement == 0 {
          svgelement = out
        }
      } else if c == '=' {
        attr := out
        for data[attr-1] >= 'A' || data[attr-1] == '-' { attr-- }
        attrname = string(data[attr:out])
      }
    }
    data[out] = c
    out++
  }
  
  // make a copy to allow memory to be freed and to insert \n at viewBox insertion point
  dt := make([]byte,out+1)
  copy(dt,data[0:svgelement])
  dt[svgelement] = '\n'
  copy(dt[svgelement+1:],data[svgelement:])
  svgelement++
  data = dt 
  
  // find node in tree to insert data, creating intermediate nodes if necessary
  a := assets
  for _, idpart := range id {
    aa := a.sub[idpart]
    if aa == nil {
      aa = &pile{sub:map[string]*pile{}}
      a.sub[idpart] = aa
    }
    a = aa
  }
  
  viewBox := toplevelmeta["viewBox"]
  if len(viewBox) < 7 {
    viewBox = fmt.Sprintf("0 0 %v %v",toplevelmeta["width"],toplevelmeta["height"])
  }
  
  ss := newSVGImageAsset(pth, viewBox, data[0:svgelement], data[svgelement:], map[string]string{})
  // At this time we do not support multiple assets with the same id. If a new asset
  // comes in with the same id it will just replace the previously stored one. We test
  // for nil here to make sure we don't replace an existing asset with nil.
  if ss != nil {
    a.asset = ss
  }
  
  addSVGSubAssets(pth, metadata, a, data[0:svgelement], data[svgelement:])
} 

// metadata contains attributes of <rect> elements within the <g> with id/label "METADATA".
// In addition to the element attributes, if the <rect> has a <desc> child, that element's
// content is stored under the name "description" in the respective map.
//
// Each rectangle describes a sub-asset to be extracted by inserting a viewBox= attribute
// between head and body (which are XML code of the SVG asset).
//
// a is the parent under which collected sub-assets are inserted into the pile.
//
// pth is the path of the main asset. It is used only in error log messages.
func addSVGSubAssets(pth string, metadata []map[string]string, a *pile, head, body []byte) {
  indexes := make([]int,0,len(metadata))
  rects := make([]*sdl.Rect,len(metadata))
  for i := range rects {
    vbox := metadata[i]["x"]+" "+metadata[i]["y"]+" "+metadata[i]["width"]+" "+metadata[i]["height"]
    r := parseViewBox(vbox)
    if r == nil {
      ShitLog = append(ShitLog, fmt.Sprintf("%v/%v: Cannot parse coordinates \"%v\"",pth,metadata[i]["id"],vbox))
    } else {
      rects[i] = r
      indexes = append(indexes, i)
    }
  }

  // sort by ascending area, i.e. rects[indexes[0]] is the largest rectangle
  sort.Slice(indexes, func(i, j int) bool { return rects[indexes[i]].W*rects[indexes[i]].H > rects[indexes[j]].W*rects[indexes[j]].H })  
  curect := &sdl.Rect{-1073741824,-1073741824,2147483647,2147483647}
  stack := []*sdl.Rect{}
  asstack := []*pile{}
  
  for {
    foundidx := -1
    for i,idx := range indexes {
      
      if idx >= 0 {
        uni := rects[idx].Union(curect)
        if uni.Equals(curect) {
          foundidx = idx
          indexes[i] = -1
          break
        }
      }
    }
    
    if foundidx < 0 {
      if len(stack) == 0 { break }
      curect = stack[len(stack)-1]
      stack = stack[0:len(stack)-1]
      a = asstack[len(asstack)-1]
      asstack = asstack[0:len(asstack)-1]
    } else {
      idpart := metadata[foundidx]["id"]
      idpart = strings.TrimRight(idpart, "0123456789")
      if idpart == "" {
        ShitLog = append(ShitLog, fmt.Sprintf("%v => rect %v: All path components must contain at least 1 non-digit character",pth, metadata[foundidx]["id"]))
      } else {
        aa := a.sub[idpart]
        if aa == nil {
          aa = &pile{sub:map[string]*pile{}}
          a.sub[idpart] = aa
        }
        asstack = append(asstack, a)
        a = aa
        stack = append(stack, curect)
        curect = rects[foundidx]
        
        var x,y int32 = 0,0 
        if len(stack) > 1 {
          x = curect.X - stack[len(stack)-1].X
          y = curect.Y - stack[len(stack)-1].Y
        }
        viewBox := fmt.Sprintf("%v %v %v %v", x, y, curect.W, curect.H)
        ss := newSVGImageAsset(pth+" => rect "+metadata[foundidx]["id"], viewBox, head, body, metadata[foundidx])
        // At this time we do not support multiple assets with the same id. If a new asset
        // comes in with the same id it will just replace the previously stored one. We test
        // for nil here to make sure we don't replace an existing asset with nil.
        if ss != nil {
          a.asset = ss
        }
      }
    }
  }
}

// Takes a string with x,y,width,height coordinates (floating point) separated by whitespace
// and/or commas and converts them into a sdl.Rect. Returns nil if there is an error.
// The unit identifier "px" may be present and will be ignored.
func parseViewBox(box string) *sdl.Rect {
  f := strings.Fields(strings.Replace(strings.Replace(box,"px","",-1),",","",-1))
  if len(f) != 4 { return nil }
  var conv [4]int32
  for i := range f {
    conv[i] = stringToInt32(f[i])
    if conv[i] == -2147483648 { return nil }
  }
  
  if conv[2] < 0 || conv[3] < 0 { return nil }
  
  return &sdl.Rect{conv[0],conv[1],conv[2],conv[3]}
}

// Converts a floating point string into an int32.
// Returns -2147483648 on error.
func stringToInt32(s string) int32 {
  num := stringToFloat64(s)
  if math.IsNaN(num) { return -2147483648 }
  return roundint32(num)
}

func stringToFloat64(s string) (res float64) {
  res = math.NaN()
  num, err := strconv.ParseFloat(s, 64)
  if err != nil { return }
  if math.IsNaN(num) { return } // ParseFloat should never produce NaN, but who knows...
  if num > 2147483647 || num < -2147483647 { return }
  return num
}

func roundint32(num float64) int32 {
  if num < 0 {
    return int32(num-.5)
  } else {
    return int32(num+.5)
  }
}

// Creates and returns a new SVGAsset,
//   errorlabel: A label used in error log entries (usually the path of the image asset)
//   vbox: a viewBox attribute value that describes the rectangle within the SVG image of the asset
//   head, body: the XML source code of the SVG file split so that inserting viewBox="<vbox>" between
//               head and body will create a valid SVG file.
//   metadata: Attributes of the <rect> that describes the asset plus optionally a "description" that
//             is taken from the <desc> element.
func newSVGImageAsset(errorlabel string, vbox string, head, body []byte, metadata map[string]string) ImageAsset {
  box := parseViewBox(vbox)
  if box == nil {
    ShitLog = append(ShitLog, fmt.Sprintf("%v: Cannot parse box coordinates \"%v\"", errorlabel, vbox))
    return nil
  }
  
  width_half := float64(box.W)/2
  height_half := float64(box.H)/2
  cxf := stringToFloat64(metadata["transform-center-x"])
  if math.IsNaN(cxf) { cxf = 0 }
  cyf := stringToFloat64(metadata["transform-center-y"])
  if math.IsNaN(cyf) { cyf = 0 }
  cx := roundint32(width_half+cxf)
  cy := roundint32(height_half-cyf)
  
  meta := util.AlmostJSON(fmt.Sprintf("%v\nx:%v\ny:%v\nwidth:%v\nheight:%v\ncenterx:%v\ncentery:%v\n",metadata["description"],box.X,box.Y,box.W,box.H,cx,cy))
  jsonMeta := map[string]interface{}{}
  err := json.Unmarshal(meta, &jsonMeta)
  if err != nil {
    ShitLog = append(ShitLog, fmt.Sprintf("%v: JSON conversion error: %v '%v'",errorlabel,err,string(meta)))
    return nil
  }
  
  return &SVGAsset{Head:head, Body:body, ViewBox: []byte("viewBox=\""+vbox+"\""), MetaJSON:meta}
}

func (a *SVGAsset) Meta(target interface{}) error {
  return json.Unmarshal(a.MetaJSON, target)
}

func (a *SVGAsset) Render(width,height int32) ([]uint32,error) {
  if width <= 0 || height <= 0 { return nil, ErrIllDimensions }

  rsvg_handle := C.rsvg_handle_new_with_flags(C.RSVG_HANDLE_FLAG_UNLIMITED|C.RSVG_HANDLE_FLAG_KEEP_IMAGE_DATA)
  if rsvg_handle == nil {
    return nil, ErrUnknown
  }
  defer C.g_object_unref(C.gpointer(rsvg_handle))

  var gerr *C.GError
  
  if len(a.Head) > 0 {
    C.rsvg_handle_write(rsvg_handle, (*C.guchar)(unsafe.Pointer(&(a.Head[0]))), C.gsize(len(a.Head)), &gerr)
    if gerr != nil {
      defer C.g_error_free(gerr)
      return nil, errors.New(C.GoString((*C.char)(gerr.message)))
    }
  }
  
  if len(a.ViewBox) > 0 {
    C.rsvg_handle_write(rsvg_handle, (*C.guchar)(unsafe.Pointer(&(a.ViewBox[0]))), C.gsize(len(a.ViewBox)), &gerr)
    if gerr != nil {
      defer C.g_error_free(gerr)
      return nil, errors.New(C.GoString((*C.char)(gerr.message)))
    }
  }
  
  if len(a.Body) > 0 {
    C.rsvg_handle_write(rsvg_handle, (*C.guchar)(unsafe.Pointer(&(a.Body[0]))), C.gsize(len(a.Body)), &gerr)
    if gerr != nil {
      defer C.g_error_free(gerr)
      return nil, errors.New(C.GoString((*C.char)(gerr.message)))
    }
  }
  
  C.rsvg_handle_close(rsvg_handle, &gerr)
  if gerr != nil {
    defer C.g_error_free(gerr)
    return nil, errors.New(C.GoString((*C.char)(gerr.message)))
  }

  data := make([]uint32, width*height)
  
  /*
  CAIRO_FORMAT_ARGB32
    each pixel is a 32-bit quantity, with alpha in the upper 8 bits,
    then red, then green, then blue.
    The 32-bit quantities are stored native-endian.
    Pre-multiplied alpha is used. (That is, 50% transparent red is 0x80800000, not 0x80ff0000.) (Since 1.0)
  */
  cairo_surface := C.cairo_image_surface_create_for_data ((*C.uchar)(unsafe.Pointer(&(data[0]))), C.CAIRO_FORMAT_ARGB32, C.int(width), C.int(height), C.int(width<<2));
  defer C.cairo_surface_destroy(cairo_surface)
  
  if C.cairo_surface_status(cairo_surface) != C.CAIRO_STATUS_SUCCESS {
    return nil, ErrUnknown
  }
  
  cr := C.cairo_create(cairo_surface)
  defer C.cairo_destroy(cr)
  
  C.rsvg_handle_render_cairo(rsvg_handle,cr)
  C.cairo_surface_flush(cairo_surface);
  
  return data,nil
}
