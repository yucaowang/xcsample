# xcsample
这是一个XuperChain的Go语言的样例代码，以存证场景为例，演示了数据如何上链和查询。

## 使用说明
Sample代码支持运行在MacOS/Linux上，需要Go 1.12+ 的运行环境

`samples`目录下是一个可执行程序，演示了数据如何上链，以及如何查询链上数据。运行该样例代码，需要先部署好XuperChain的网络，此处以本地节点为例，在本地运行一个超级链节点，默认的为 `localhost:37101`，配置在代码中。

编译运行方法：

```
sh build.sh
./xcsample
```