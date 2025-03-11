module github.com/Krchnk/gw-currency-wallet

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/proto-exchange/exchange_grpc v0.0.0 // Укажите правильную версию после генерации proto
    google.golang.org/grpc v1.62.0
    github.com/sirupsen/logrus v1.9.3
    github.com/patrickmn/go-cache v2.1.0+incompatible
    github.com/dgrijalva/jwt-go v3.2.0+incompatible
    golang.org/x/crypto v0.20.0
    github.com/lib/pq v1.10.9
)
