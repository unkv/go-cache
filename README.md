# go-cache
 
Modified from <https://github.com/fanjindong/go-cache> with Generics in Go 1.18+.

## Install

`go get -u gopkg.in/unkv/go-cache.v0`

## Basic Usage

```go
import "gopkg.in/unkv/go-cache.v0"

func main() {
    c := cache.NewWithStrKey[int]()
    c.Set("a", 1)
    c.Set("b", 1, cache.WithEx[string, int](1*time.Second))
    time.Sleep(1*time.Second)
    c.Get("a") // 1, true
    c.Get("b") // 0, false
}
```