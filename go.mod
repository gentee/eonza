module eonza

go 1.16

require (
	github.com/PuerkitoBio/goquery v1.8.0
	github.com/alecthomas/chroma v0.9.4
	github.com/atotto/clipboard v0.1.4
	github.com/boombuler/barcode v1.0.1 // indirect
	github.com/gentee/eonza-pro v0.0.0-00010101000000-000000000000
	github.com/gentee/gentee v1.21.0
	github.com/gentee/systray v1.3.1
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/kataras/golog v0.1.7
	github.com/kr/text v0.2.0 // indirect
	github.com/labstack/echo/v4 v4.6.1
	github.com/lib/pq v1.10.4
	github.com/pquerna/otp v1.3.0 // indirect
	github.com/robfig/cron/v3 v3.0.1
	github.com/xhit/go-simple-mail/v2 v2.10.0
	github.com/xuri/excelize/v2 v2.4.1
	github.com/yuin/goldmark v1.4.4
	github.com/yuin/goldmark-highlighting v0.0.0-20210516132338-9216f9c5aa01
	golang.org/x/crypto v0.0.0-20211115234514-b4de73f9ece8
	golang.org/x/sys v0.0.0-20211116061358-0a5406a5449c
	golang.org/x/text v0.3.7
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/ini.v1 v1.64.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/gentee/eonza-pro => ../eonza-pro
