FROM scratch

WORKDIR $GOPATH/src/github.com/hezhizheng/go-wxpush
COPY ./go-wxpush_linux_amd64 $GOPATH/src/github.com/hezhizheng/go-wxpush

ENTRYPOINT ["./go-wxpush_linux_amd64"]
 # 默认参数
CMD ["-port", "5566", "-appid", "","-secret","","-userid","","-template_id","","-base_url","https://push.hzz.cool/detail","-title","","-content","","-tz",""]