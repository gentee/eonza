module eonza

go 1.16

require (
	github.com/PuerkitoBio/goquery v1.8.0
	github.com/alecthomas/chroma v0.10.0
	github.com/atotto/clipboard v0.1.4
	github.com/boombuler/barcode v1.0.1 // indirect
	github.com/gentee/eonza-pro v0.0.0-00010101000000-000000000000
	github.com/gentee/gentee v1.22.0
	github.com/gentee/systray v1.3.1
	github.com/go-sql-driver/mysql v1.6.0
	github.com/go-test/deep v1.0.8 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/gorilla/websocket v1.5.0
	github.com/kataras/golog v0.1.7
	github.com/kr/text v0.2.0 // indirect
	github.com/labstack/echo/v4 v4.7.2
	github.com/lib/pq v1.10.6
	github.com/pquerna/otp v1.3.0 // indirect
	github.com/robfig/cron/v3 v3.0.1
	github.com/xhit/go-simple-mail/v2 v2.11.0
	github.com/xuri/excelize/v2 v2.6.0
	github.com/yuin/goldmark v1.4.12
	github.com/yuin/goldmark-highlighting v0.0.0-20220208100518-594be1970594
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a
	golang.org/x/text v0.3.7
	golang.org/x/time v0.0.0-20220411224347-583f2d630306 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/ini.v1 v1.66.5
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/gentee/eonza-pro => ../eonza-pro
