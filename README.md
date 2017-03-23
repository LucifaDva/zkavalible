package zkavalible
============

Introduce
---------------
基于Zookeeper的高可用性保证，利用Zookeeper临时节点概念，即一旦进程退出，临时节点就会消失。

根据可监视客户端ZooKeeper ZNode的状态、内容等的改变，从而判断进程是否已经退出了。按照Example方式，在需要高可用行的进程中加入代码，这样多个进程就可以进行高可用行协调了。

（1）互斥型服务

* 只允许一个进程运行，如一些需要独占资源的进程
* 保证该进程一定运行，并且保证有一个进程运行时其他进程无效不执行逻辑
    
（2）无数量限制服务

* 允许无数个进程运行

（3）允许有限数量服务

* 允许N个进程同时运行，当进程数量超过限制数量N时，之后启动的进程无效不执行逻辑

Example
---------------
```GO
package main
import(
    "zkavalible"
	"fmt"
)
func main(){
	zka, err := zkavalible.New("/zkavailable/process/", "process1", []string{"127.0.0.1:2181"}, 1)
	if err != nil {
		fmt.Println("new zkavalible err: ", err)
	}
	go zka.Start()
	for {
		a := <-zka.RunChan
		fmt.Println("signal:", a)
		//break
	}
	//zka.Exit()
	fmt.Println("exit zkavalible")
}

```


