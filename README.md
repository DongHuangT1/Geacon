# Geacon

*本项目仅限于安全研究和教学，严禁用于非法用途！*

## Usage

1. 修改`core/profile.go`中的配置信息
2. 设置平台和架构`set "CGO_ENABLED=0" && set "GOOS=linux" && set "GOARCH=amd64"`
3. 编译生成`go build -trimpath -ldflags="-s -w" main.go`

## Screenshot

![](https://i.loli.net/2021/10/20/n3oKctpNRy29G4T.jpg)

## Reference

- https://github.com/darkr4y/geacon
- https://wbglil.gitbook.io/cobalt-strike

## License

[GPL 3.0](https://github.com/DongHuangT1/Geacon/blob/master/LICENSE)
