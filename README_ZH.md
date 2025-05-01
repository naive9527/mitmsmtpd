**其他语言版本: [English](README.md), [中文](README_zh.md).**

## 启动服务
go build .  # 编译出 mitmsmtpd 可执行文件  
将 config_sample.yaml 复制为 config.yaml 并修改成自己的配置
./mitmsmtpd   # 启动服务

## 业务流程

为了方便描述假设信息如下：

邮件服务器: smtp.example.com
邮件用户: user01@example.com,user02@example.com
MUA的IP： 10.10.20.10 
mitmsmtpd的IP： 10.10.20.111

### 一般正常情况
    1、user01@example.com 在 10.10.20.10编写邮件设置收信人为  user02@example.com 
    2、登录 smtp.example.com成功
    3、投递邮件成功

### 使用mitmsmtpd后
    1、在 10.10.20.10 修改 smtp.example.com的dns解析为 mitmsmtpd的IP （或者直接指定smtp的地址为mitmsmtpd的IP）
    2、user01@example.com 在 10.10.20.10 编写邮件设置收信人为  user02@example.com 
    3、登录 mitmsmtpd 成功
    4、mitmsmtpd 根据规则校验邮件是否符合标准，不符合则返回错误信息，符合进行下一步
    5、mitmsmtpd 使用发信的用户名和密码 登录 真实的SMTP服务器（根据发信人user01@example.com和config.yaml进行查询到）
    6、登录失败则返回错误信息，成功则继续下一步
    7、mitmsmtpd 发送邮件
    8、如果某一步失败了，会返回错误信息，并将整个邮件保存为eml邮件文件，并且通知管理员(目前实现了邮件通知)

## TLS配置
    ### 使用真实的TLS证书和私钥来保护SMTP服务。
    自行申请即可

    ### 使用自签名证书 如cfssl等
    生成私有ca, 将ca配置到 MUA 上面 
    使用私有ca 生成证书配置到 mitmsmtpd 上面 



