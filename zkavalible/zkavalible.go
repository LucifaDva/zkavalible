package zkavalible
import (
    "fmt"
    "os"
    "net"
    "strings"
    "errors"
    "github.com/samuel/go-zookeeper/zk"
    "zookeeper"
    "time"
)
/* 使用包方法——
	（1）import zkavalible包
	（2）在进程逻辑代码之前，添加如下逻辑
	zka, err := zkavalible.New("/zkavailable/process/", "process1", []string{"103.28.10.236:2181"}, 0)
	if err != nil {
		fmt.Println("new zkavalible err: ", err)
	}
	go zka.Start()
	for {
		a := <-zka.RunChan
		fmt.Println("signal:", a)
		break
	}
	zka.Exit()
	fmt.Println("exit zkavalible")
 */

type avalZkStruct struct {
	RootPath 			string 		//process root path
	Name 			string
	Servers 		[]string
	Zookeeper 		*zookeeper.Zookeeper
	ServiceNum      int       //1：互斥只能有一个在运行；N>1：允许两个运行；0：不限
	RunChan   		chan int
	exit 			bool
}

func New(rootPath, name string, servers []string, serviceNum int) (a *avalZkStruct, err error) {
	if serviceNum < 0 {
		err = errors.New("serviceNum is invalide")
		return
	}
	zookeeper, err := zookeeper.New(servers)
	if err != nil {
		return
	}
	runchan := make(chan int)
	return &avalZkStruct{RootPath:rootPath, Name:name, Servers:servers, Zookeeper:zookeeper, ServiceNum:serviceNum, RunChan:runchan, exit:false}, nil
}

func (a *avalZkStruct) Exit() {
	a.exit = true
}

func (a *avalZkStruct) startProcess() (err error){
	pathName := a.RootPath + a.Name
	lock := a.Zookeeper.NewLock(pathName)
	lock.Lock()
	if a.ServiceNum == 1 {	//互斥类型的服务
		isExists, err := a.Zookeeper.Exists(pathName)
		echoLog("[INFO]", fmt.Sprintf("互斥型服务:%v pathName:%v isExists: %v, err: %v", a.Name, pathName, isExists, err))
		if err == nil && isExists == true {	//已经一个进程在运行，本服务无法运行
			echoLog("[INFO]", fmt.Sprintf("互斥型服务:%v 已经在运行", a.Name))
			lock.Unlock()
			return err
		}
		a.RunChan <- 1
		zkData := a.generateZkData()
		err = a.Zookeeper.ZNodeCreate(pathName, zkData)
		lock.Unlock()
		return err
	} else if a.ServiceNum == 0 {    //无数量限制类型服务
		echoLog("[INFO]", fmt.Sprintf("无数量限制类型服务:%v", a.Name))
		a.RunChan <- 1
		zkData := a.generateZkData()
		err = a.Zookeeper.ZNodeCreateSeq(pathName, zkData)
		lock.Unlock()
		return err
	} else if a.ServiceNum > 1 {	    //有数量限制类型服务
		echoLog("[INFO]", fmt.Sprintf("有数量限制类型服务:%v", a.Name))
		lockpath := a.Name + "lock"
		count := 0
		children, _ := a.Zookeeper.Children("/sang")
		for _, p := range children {
			fmt.Println(fmt.Sprintf("all child: %v, p: %v", children, p))
			if strings.Contains(p, pathName) || p != lockpath {
				count = count + 1
			}
		}
		fmt.Println("children count: ", count)
		if count >= a.ServiceNum {   //已经达到了数量限制
			echoLog("[WARN]", fmt.Sprintf("有数量限制类型服务:%v 已经达到进程最大限制:%v", a.Name, a.ServiceNum))
			lock.Unlock()
			return err
		}
		a.RunChan <- 1
		zkData := a.generateZkData()
		err = a.Zookeeper.ZNodeCreateSeq(pathName, zkData)
		lock.Unlock()
		return err
	}
	return 
}

func (a *avalZkStruct) generateZkData() (zkData []byte) {
	pathName := a.RootPath + a.Name
	ip, _ := getLocalIp()
	zkDataStr := fmt.Sprintf("%v_%v", ip, os.Getpid())
	echoLog("[INFO]", fmt.Sprintf("generateZkData pathName: %v, zkDatastr: %v", pathName, zkDataStr))
	zkData = []byte(zkDataStr)
	return
}



func (a *avalZkStruct) Start() (err error){
	a.startProcess()
	a.watchProcess()
	return
}


func (a *avalZkStruct) watchProcess() {
	pathName := a.RootPath + a.Name
	var count int
	if a.ServiceNum == 1 {
		for {
			fmt.Println("--------------")
			_, event, err := a.Zookeeper.ExistsW(pathName)
			if err != nil {
				echoLog("[ERROR]", fmt.Sprintf("ExistsW %v err: %v", pathName, err))
				continue
			}
			if a.exit == true {
				return
			}
			e := <- event
			if e.Type == zk.EventNodeDeleted {
				lock := a.Zookeeper.NewLock(pathName)
				if a.ServiceNum == 1 {
					lock.Lock()
					isExists, err := a.Zookeeper.Exists(pathName)
					echoLog("[INFO]", fmt.Sprintf("有崩溃后节点检查 pathName: %v, isExists: %v, err: %v", pathName, isExists, err))
					if isExists == false && err == nil {
						echoLog("[INFO]", fmt.Sprintf("启动服务: %v", a.Name))
						a.RunChan <- 1
					} else {
						echoLog("[INFO]", fmt.Sprintf("不做操作: %v", a.Name))
					}
					zkData := a.generateZkData()
					err = a.Zookeeper.ZNodeCreate(pathName, zkData)
					lock.Unlock()
				}
			}
		}
	} else if a.ServiceNum == 0 {
		for {
			fmt.Println("--------------")
			count = 0
			_, event, err := a.Zookeeper.ChildrenW(a.RootPath)
			if err != nil {
				echoLog("[ERROR]", fmt.Sprintf("ExistsW %v err: %v", pathName, err))
				continue
			}
			if a.exit == true {
				return
			}
			e := <- event
			if e.Type == zk.EventNodeChildrenChanged {
				children, _ := a.Zookeeper.Children(a.RootPath)
				for _, p := range children {
					if strings.Contains(p, pathName) {
						count = count + 1
					}
				}
			}
		}
	} else if a.ServiceNum > 1 {
		for {
			fmt.Println("--------------")
			count = 0
			_, event, err := a.Zookeeper.ChildrenW(a.RootPath)
			if err != nil {
				echoLog("[ERROR]", fmt.Sprintf("ExistsW %v err: %v", pathName, err))
				continue
			}
			if a.exit == true {
				return
			}
			e := <- event
			if e.Type == zk.EventNodeChildrenChanged {
				lock := a.Zookeeper.NewLock(a.RootPath)
				children, _ := a.Zookeeper.Children(a.RootPath)
				for _, p := range children {
					if strings.Contains(p, pathName) {
						count = count + 1
					}
				}
				if count < a.ServiceNum {
					a.RunChan <- 1
					zkData := a.generateZkData()
					err = a.Zookeeper.ZNodeCreateSeq(pathName, zkData)
					echoLog("[INFO]", fmt.Sprintf("进程不够了，阻塞的进程可以运行了: %v", a.Name))
				} else if count == 0 {
					fmt.Println("已经没有进程在运行了，报警")
				}
				lock.Unlock()
			}
		}
	}

}




func getLocalIp() (ip string, err error) {
	ip = "127.0.0.1"
	addrs, err := net.InterfaceAddrs()
    if err != nil {
    	fmt.Println("err:", err)
    	return
    }
    if len(addrs) > 1 {
    	ipstr := fmt.Sprintf("%s", addrs[1])
    	arr := strings.Split(ipstr, "/")
    	ip = arr[0]
    }
    return
}

func echoLog(level, content string) {
	outputContent := fmt.Sprintf("%s\t[%s]\t%s", time.Now().Format("2006-01-02 15:04:05"), level, content)
	fmt.Println(outputContent)
}

