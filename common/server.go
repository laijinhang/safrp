package common

type Server interface {
	ReadServer(ctx *Context, funs []func(ctx *Context))
	SendServer(ctx *Context, funs []func(ctx *Context))
	Protocol() string
}

type TCPServer struct {
	IP string
	Port string
}

func (this *TCPServer) ReadServer(ctx *Context, funs []func(ctx *Context)) {
	ctx.IP = this.IP
	ctx.Port = this.Port
	for i := 0;i < len(funs);i++ {
		funs[i](ctx)
	}
}

func (this *TCPServer) SendServer(ctx *Context, funs []func(ctx *Context)) {
	ctx.IP = this.IP
	ctx.Port = this.Port
	for i := 0;i < len(funs);i++ {
		funs[i](ctx)
	}
}

func (t *TCPServer) Protocol() string {
	return "tcp"
}

type UDPServer struct {
	IP string
	Port string
}

func (this *UDPServer) ReadServer(ctx *Context, funs []func(ctx *Context)) {
	ctx.IP = this.IP
	ctx.Port = this.Port
	for i := 0;i < len(funs);i++ {
		funs[i](ctx)
	}
}

func (this *UDPServer) SendServer(ctx *Context, funs []func(ctx *Context)) {
	ctx.IP = this.IP
	ctx.Port = this.Port
	for i := 0;i < len(funs);i++ {
		funs[i](ctx)
	}
}

func (this *UDPServer) Protocol() string {
	return "udp"
}

type HTTPServer struct {
	IP string
	Port string
}

func (this *HTTPServer) ReadServer(ctx *Context, funs []func(ctx *Context)) {
	ctx.IP = this.IP
	ctx.Port = this.Port
	for i := 0;i < len(funs);i++ {
		funs[i](ctx)
	}
}

func (this *HTTPServer) SendServer(ctx *Context, funs []func(ctx *Context)) {
	ctx.IP = this.IP
	ctx.Port = this.Port
	for i := 0;i < len(funs);i++ {
		funs[i](ctx)
	}
}

func (this *HTTPServer) Protocol() string {
	return "http"
}
