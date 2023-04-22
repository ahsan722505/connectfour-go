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
	photo string
}

type UserClient struct{
	Username string 
	UserId string 	
	Host bool
	GameId int
	Photo string 	
}
type Room struct{
	users []User
	playAgainRequest bool
}

type Packet struct {
	Type string
	Data interface {}
}

var rooms = make(map[string]Room)
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
		photo :=data["photo"].(string)
		userId :=data["userId"].(string)
		user := User{
			conn: ws,
			username : username,
			host: true,
			userId:  userId,
			gameId: 1,
			photo: photo,
		}
		users := []User{user}
		rooms[roomId]= Room{
			users: users,
			playAgainRequest: false,
		}
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
		roomUsers := rooms[roomId].users
		for _,user := range roomUsers{
			if user.userId == oppId{
				packet.Type="oppturn"
				packet.Data = cellInd
				user.conn.WriteJSON(packet)
			}
		}
		
	}
	if packet.Type == "leaveGame"{
		data := packet.Data.(map[string] interface{})
		roomId := data["roomId"].(string)
		oppId := data["oppId"].(string)
		roomUsers := rooms[roomId].users
		for _,user := range roomUsers{
			if user.userId == oppId{
				packet.Data = nil
				user.conn.WriteJSON(packet)
			}
		}
		
	}

	if packet.Type == "playAgainRequest"{
		data := packet.Data.(map[string] interface{})
		roomId := data["roomId"].(string)
		oppId := data["oppId"].(string)
		room := rooms[roomId]
		println("playAgainRequest")
		roomUsers := room.users
		if room.playAgainRequest{
			for _,user := range roomUsers{
				packet.Type="playAgain"
				packet.Data = nil
				user.conn.WriteJSON(packet)
			}
			room.playAgainRequest = false
			rooms[roomId] = room
		}else{
			// forwarding request to other client	
			for _,user := range roomUsers{
				if user.userId == oppId{
					packet.Data = nil
					user.conn.WriteJSON(packet)
				}
			}
			room.playAgainRequest = true
			rooms[roomId] = room
		}
		
	}

	if packet.Type == "join-room"{
		data := packet.Data.(map[string] interface{})
		roomId := data["roomId"].(string)
		username := data["username"].(string)
		userId := data["userId"].(string)
		photo := data["photo"].(string)
		user := User{
			conn : ws,
			username: username,
			userId: userId,
			host: false,
			gameId: 2,
			photo: photo,
		}
		room := rooms[roomId]
		roomUsers := room.users
		roomUsers = append(roomUsers,user)
		room.users = roomUsers
		rooms[roomId]=room
		for i := 0 ; i< len(roomUsers) ; i++{
			socket := roomUsers[i].conn
			packet.Type="start-game"
			players := []UserClient{}
			for j:= 0 ; j < len(roomUsers) ; j++{
				players=append(players, UserClient{
					Username: roomUsers[j].username,
					UserId: roomUsers[j].userId,
					Host: roomUsers[j].host,
					GameId: roomUsers[j].gameId,
					Photo: roomUsers[j].photo,
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