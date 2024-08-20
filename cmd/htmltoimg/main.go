package main

import (
    "os"

    "github.com/go-rod/rod"
    "github.com/go-rod/rod/lib/proto"
    "github.com/ysmood/gson"
)

var (
	html = ``
)

func main() {
    browser := rod.New()
    browser.ControlURL("ws://chrome:3000")
    if err := browser.Connect(); err != nil {
        panic(err)
    }

    buf, err := browser.MustPage("file:///var/opt/test.html").Screenshot(true, &proto.PageCaptureScreenshot{
        Format:  proto.PageCaptureScreenshotFormatJpeg,
        Quality: gson.Int(90),
    })
    if err != nil {
        panic(err)
    }

    err = os.WriteFile("/var/opt/out.png", buf, 0644)
    if err != nil {
        panic(err)
    }
}
