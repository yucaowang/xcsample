package client

import (
	"errors"
	crypto_client "github.com/xuperchain/xuperunion/crypto/client"
	crypto_base "github.com/xuperchain/xuperunion/crypto/client/base"
	"github.com/xuperchain/xuperunion/pb"
	"google.golang.org/grpc"
	"sync"
	// "xchainsdk/pb"
)

var (
	//定义客户端的返回的错误类型，作为PB错误类型不足的补充， 后面可以逐步移动到pb文件里面去
	ExtraErrInvalidAddress   = errors.New("Invalid address")
	ExtraErrNegativeAmount   = errors.New("Amount in transaction can not be negative number")
	ExtraErrInvalidAmount    = errors.New("Invalid amount number")
	ExtraErrOverloadNotifier = errors.New("Too many notifiers to watch")
)

/**
 * @filename client.go
 * @desc
 * @create time 2018-09-03 10:43:26
**/
type Notifier struct {
	tx *pb.TxStatus
	qc int // querying count
	n  INotifier
}
type XClient struct {
	pb.XchainClient
	conn         *grpc.ClientConn
	host         string
	version      string
	address      string
	publicKey    string
	privateKey   string
	cryptoType   string
	cryptoClient crypto_base.CryptoClient
	notifierChan chan *Notifier
	stop         chan bool
	wg, wgNi     sync.WaitGroup
}

const CHAN_SIZE = 10000

type XCOption func(*XClient)

func WithVersion(version string) XCOption {
	return func(c *XClient) {
		c.version = version
	}
}
func WithAddress(addr string) XCOption {
	return func(c *XClient) {
		c.address = addr
	}
}
func WithPublicKey(k string) XCOption {
	return func(c *XClient) {
		c.publicKey = k
	}
}
func WithPrivateKey(sk string) XCOption {
	return func(c *XClient) {
		c.privateKey = sk
	}
}
func WithCryptoType(ct string) XCOption {
	return func(c *XClient) {
		c.cryptoType = ct
	}
}
func NewXClientWithOpts(host string, ops ...XCOption) (*XClient, error) {
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	c := pb.NewXchainClient(conn)
	client := &XClient{
		conn:         conn,
		XchainClient: c,
		host:         host,
		notifierChan: make(chan *Notifier, CHAN_SIZE),
		stop:         make(chan bool, 1),
		cryptoType:   crypto_client.CryptoTypeDefault,
	}
	for _, op := range ops {
		op(client)
	}
	// client sdk using crypto type in public key (curve name)
	xcc, err := crypto_client.CreateCryptoClientFromJSONPublicKey([]byte(client.publicKey))
	if err != nil {
		return nil, err
	}
	client.cryptoClient = xcc
	return client, nil
}

// 设置grpc的客户端参数, 不够优雅, 需要将之前的链接断开并重新链接
func (xc *XClient) SetClientGrpcOpts(host string, gOps ...grpc.DialOption) error {
	xc.conn.Close()
	conn, err := grpc.Dial(host, gOps...)
	if err != nil {
		return err
	}
	c := pb.NewXchainClient(conn)
	xc.conn = conn
	xc.XchainClient = c
	return nil
}

//退出监听： force=false表示等待所有已有的回调处理完 再退出，否则就是强制退出,已有未回调的将会丢失
func (c *XClient) Stop(force bool) {
	if !force {
		c.wgNi.Wait()
	}
	c.wg.Wait()
	c.stop <- true
	//关闭链接
	if c.conn != nil {
		c.conn.Close()
	}
}
func (c *XClient) GetAddress() string {
	return c.address
}
func (c *XClient) GetPublicKey() string {
	return c.publicKey
}
func (c *XClient) GetPrivateKey() string {
	return c.privateKey
}
func (c *XClient) SetAddress(addr string) {
	c.address = addr
}
func (c *XClient) SetPublicKey(pk string) {
	c.publicKey = pk
}
func (c *XClient) SetPrivateKey(sk string) {
	c.privateKey = sk
}
