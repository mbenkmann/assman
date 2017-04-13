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

package main

import (
  "os"
  "fmt" 
  "sort"
  "image"
  "image/png"
  "strings"
  
  "../ass"
)

func main() {
  // Scan all directories recursively from the current directory down
  ass.Add(".")
  assets := ass.List("/")
  sort.Strings(assets)
  for _, a := range assets {
    var meta map[string]interface{}
    err := ass.Meta(a,&meta)
    if err != nil { panic(err) }
    var width, height int = int(meta["width"].(float64)), int(meta["height"].(float64))
    img, err := ass.Image(a, width, height)
    if err != nil { panic(err) }
    if img == nil { panic("img is nil") }
    if len(img) != width*height { panic(fmt.Sprintf("image size is %v instead of %v", len(img), width*height)) }
    fmt.Printf("-> %v (%v)\n",a,meta)
    rgba := image.NewRGBA(image.Rect(0,0,width,height))
    if rgba.Stride != width<<2 { panic("unsupported stride") }
    for i, pixel := range img {
      rgba.Pix[i<<2+2] = uint8(pixel) // B
      pixel >>= 8
      rgba.Pix[i<<2+1] = uint8(pixel) // G
      pixel >>= 8
      rgba.Pix[i<<2+0] = uint8(pixel) // R
      pixel >>= 8
      rgba.Pix[i<<2+3] = uint8(pixel) // A
    }
    
    fname := strings.Replace(a, "/", "_",-1)+".png"
    f, err := os.Create(fname)
    if err != nil {
      panic(err)
    }
    defer f.Close()
    if err := png.Encode(f, rgba); err != nil {
      panic(err)
    }
  }
  fmt.Println(strings.Join(ass.ShitLog,"\n"))
}

