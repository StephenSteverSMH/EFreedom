package message

const (
	DefaultMessageSize = 4096
	DefaultMessageChanSize = 128
)

// 单条数据消息
type Message struct {
	Data []byte
}

// 数据池，控制并发
type MessagePool struct {
	messageChan chan *Message
}

// 获取消息，如果管道消息用完，则会堵塞
func GetMessage(pool *MessagePool) *Message{
	return <-pool.messageChan
}

// 使用完消息，放回消息
func PutMessage(msg *Message, pool *MessagePool){
	pool.messageChan <- msg
}

// 创建数据池
func CreatePool(messageSize, messageChanSize int) MessagePool{
	pool := MessagePool{
		messageChan: make(chan *Message, messageChanSize),
	}
	for i:=0;i<messageChanSize;i++{
		msg := Message{
			Data: make([]byte, messageSize),
		}
		pool.messageChan <- &msg
	}
	return pool
}

// 清空消息
func EmptyMessage(msg *Message, messageSize int){
	msg.Data = make([]byte, messageSize)
}