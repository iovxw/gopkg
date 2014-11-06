GoPKG
=====

Golang package manager

Use:
----

`go get github.com/Bluek404/gopkg`

需要在项目根目录建立一个`gopkg.yaml`

内容为

```yaml
packages:
  自定义package名称: package路径
```

例如：

```yaml
packages:
  yaml:gopkg.in/yaml.v2
  macaron: github.com/Unknwon/macaron
  socket:code.google.com/p/go.net/websocket
```

然后运行`gopkg get`

import时就可以：

```go
import (
	"yaml"
	"macaron"
	"socket"
)
```

当运行`gopkg get`时会生成一个*packages*文件夹，并把get的package软链接过去

如果需要IDE支持gopkg下载的package的话，需要把生成的*packages*文件夹路径添加到*GOPATH*中

通过`gopkg`可以运行所有`go`的命令，而且会自动添加*packages*文件夹到*GOPATH*

比如通过`gopkg build`编译的话，就不需要提前把*packages*添加到GOPATH了

