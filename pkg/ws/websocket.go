package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gorilla/websocket"
)
type User struct{
	conn  *websocket.Conn
	username string
	userId string
	host bool
	gameId int
}

type UserClient struct{
	Username string 
	UserId string 	
	Host bool
	GameId int 	
}

type Packet struct {
	Type string
	Data interface {}
}

var rooms = make(map[string][]User)
var upgrader= websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {return true},
}

func handleConnection(w http.ResponseWriter , r *http.Request){
		ws,err := upgrader.Upgrade(w,r,nil)
		if err != nil{
			log.Println(err)
		}
		log.Println("client connected")
		receiver(ws)
}

func receiver(ws *websocket.Conn){
	for{
		packet := &Packet{}
		err :=ws.ReadJSON(packet)
	if err != nil{
		log.Println(err)
	}
	if packet.Type == "create-room"{
		uuid,_ := exec.Command("uuidgen").Output()
		roomId :=strings.TrimSpace(string(uuid))
		data := packet.Data.(map[string] interface {})
		username :=data["username"].(string)
		userId :=data["userId"].(string)
		user := User{
			conn: ws,
			username : username,
			host: true,
			userId:  userId,
			gameId: 1,
		}
		users := []User{user}
		rooms[roomId]= users
		packet.Type="room-created"
		packet.Data= roomId
		log.Println(packet)
		ws.WriteJSON(packet)
	}

	if packet.Type == "myturn"{
		data := packet.Data.(map[string] interface{})
		cellInd := data["cellInd"].(float64)
		roomId := data["roomId"].(string)
		oppId := data["oppId"].(string)
		room := rooms[roomId]
		for _,user := range room{
			if user.userId == oppId{
				packet.Type="oppturn"
				packet.Data = cellInd
				user.conn.WriteJSON(packet)
			}
		}
		
	}
	if packet.Type == "playAgainSignal" || packet.Type == "leaveGame"{
		data := packet.Data.(map[string] interface{})
		roomId := data["roomId"].(string)
		oppId := data["oppId"].(string)
		room := rooms[roomId]
		for _,user := range room{
			if user.userId == oppId{
				packet.Data = nil
				user.conn.WriteJSON(packet)
			}
		}
		
	}

	if packet.Type == "join-room"{
		data := packet.Data.(map[string] interface{})
		roomId := data["roomId"].(string)
		username := data["username"].(string)
		userId := data["userId"].(string)
		user := User{
			conn : ws,
			username: username,
			userId: userId,
			host: false,
			gameId: 2,
		}
		room := rooms[roomId]
		room = append(room,user)
		rooms[roomId]=room
		for i := 0 ; i< len(room) ; i++{
			socket := room[i].conn
			packet.Type="start-game"
			players := []UserClient{}
			for j:= 0 ; j < len(room) ; j++{
				players=append(players, UserClient{
					Username: room[j].username,
					UserId: room[j].userId,
					Host: room[j].host,
					GameId: room[j].gameId,
				})
			}
			packet.Data=players
			jsonData,err := json.Marshal(packet)
			if err != nil{
				fmt.Println(err)
			}
			fmt.Printf("json data: %s\n", jsonData)
			socket.WriteJSON(packet)
		}
	}
		


	
}
}

func setUpRoutes(){
	http.HandleFunc("/ws",handleConnection)
}
func StartWebSocketServer(){
	setUpRoutes()
	http.ListenAndServe(":4000",nil)
}