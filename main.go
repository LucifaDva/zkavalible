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
		break
	}
	zka.Exit()
	fmt.Println("exit zkavalible")
}
