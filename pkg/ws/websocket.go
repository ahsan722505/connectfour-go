package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)
type User struct{
	conn  *websocket.Conn
	username string
	userId string
	gameId int
	photo string
	disconnected bool
}

type UserClient struct{
	Username string 
	UserId string 	
	GameId int
	Photo string 	
}
type Room struct{
	users []User
	playAgainRequest bool
	boardState []interface{}
	startTime string
	currentPlayer float64
}

type Packet struct {
	Type string
	Data interface {}
}

var rooms = make(map[string]Room)
var clients = make(map[*websocket.Conn]string)
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
func checkConnection(ws *websocket.Conn){
	room := rooms[clients[ws]]
	roomUsers := room.users
	for i := range roomUsers{
		if roomUsers[i].conn == ws && roomUsers[i].disconnected{
			// emit leave game to users in this room
			packet := &Packet{}
			packet.Type = "leaveGame"
			packet.Data = nil
			for j := range roomUsers{
				if roomUsers[j].conn != ws{
					roomUsers[j].conn.WriteJSON(packet)
				}
			}
			cleanupResources(clients[ws])
		}
	}
}
func cleanupResources(roomId string){
	delete(rooms,roomId)
}

func receiver(ws *websocket.Conn){
	for{
		packet := &Packet{}
		// fmt.Println(ws)
		err :=ws.ReadJSON(packet)
	if err != nil{
		log.Println(err)
		fmt.Println("error in reading packet")
		room := rooms[clients[ws]]
		roomUsers := room.users
		for i := range roomUsers{
			if roomUsers[i].conn == ws{
				roomUsers[i].disconnected = true
				room.users = roomUsers
				rooms[clients[ws]] = room
				time.AfterFunc(6*time.Second, func(){
						checkConnection(ws)
				})
			}
		}
		break
	}
	if packet.Type == "create-room"{
		uuid,_ := exec.Command("uuidgen").Output()
		roomId :=strings.TrimSpace(string(uuid))
		clients[ws]=roomId
		rooms[roomId]= Room{
			users: []User{},
			playAgainRequest: false,
		}
		packet.Type="room-created"
		packet.Data= roomId
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
		fmt.Println("leaveGame")
		data := packet.Data.(map[string] interface{})
		roomId := data["roomId"].(string)
		oppId := data["oppId"].(string)
		roomUsers := rooms[roomId].users
		fmt.Println(oppId)
		for _,user := range roomUsers{
			if user.userId == oppId{
				packet.Data = nil
				user.conn.WriteJSON(packet)
			}
		}
		cleanupResources(roomId)	
	}

	if packet.Type == "sendMessage"{
		fmt.Println("sendMessage");
		data := packet.Data.(map[string] interface{})
		roomId := data["roomId"].(string)
		oppId := data["oppId"].(string)
		message := data["message"].(string)
		roomUsers := rooms[roomId].users
		for _,user := range roomUsers{
			if user.userId == oppId{
				packet.Type="receiveMessage"
				packet.Data = message
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
			board := make([]interface{}, 6)
			for i := 0; i < 6; i++ {
				board[i] = make([]int, 6)
			}
			room.boardState = board
			room.startTime = time.Now().Format("2006-01-02 15:04:05")
			room.currentPlayer = 1
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
	if packet.Type == "saveState"{
		data := packet.Data.(map[string] interface{})
		roomId := data["roomId"].(string)
		boardState := data["boardState"].([]interface{})
		startTime := data["startTime"].(string)
		currentPlayer := data["currentPlayer"].(float64)
		room := rooms[roomId]
		if len(boardState) > 0{
			room.boardState = boardState
		}
		if startTime != ""{
			room.startTime = startTime
		}
		room.currentPlayer = currentPlayer
		rooms[roomId] = room
	}

	if packet.Type == "join-room"{
		data := packet.Data.(map[string] interface{})
		roomId := data["roomId"].(string)
		username := data["username"].(string)
		userId := data["userId"].(string)
		photo := data["photo"].(string)
		room := rooms[roomId]
		// if room doesnot exist send bad request event
		if room.users == nil{
			packet.Type="badRequest"
			packet.Data = nil
			ws.WriteJSON(packet)
			break
		}else{
			packet.Type="validRoom"
			packet.Data = nil
			ws.WriteJSON(packet)
		}


	
		clients[ws]=roomId
		roomUsers := room.users
		fmt.Println(userId,roomId)

		// checking if user is trying to reconnect
		found := false
		for i := range roomUsers{
			if roomUsers[i].userId == userId{
				roomUsers[i].conn = ws
				room.users = roomUsers
				rooms[roomId] = room
				found = true
				break
			}
		}
		if found{
			// set disconnected to false
			for i := range roomUsers{
				if roomUsers[i].userId == userId{
					roomUsers[i].disconnected = false
					room.users = roomUsers
					rooms[roomId] = room
					break
				}
			}
			if len(room.boardState) > 0{
			packet.Type="syncState"
			packet.Data = map[string]interface{}{
				"boardState": room.boardState,
				"startTime": room.startTime,
				"currentPlayer": room.currentPlayer,
			}
			fmt.Println("sending syncState")
			fmt.Println(packet.Data)
			ws.WriteJSON(packet)
		}
			continue
		}

		user := User{
			conn : ws,
			username: username,
			userId: userId,
			gameId: len(roomUsers) + 1,
			photo: photo,
		}
		roomUsers = append(roomUsers,user)
		room.users = roomUsers
		rooms[roomId]=room
		if len(roomUsers) == 2{
		for i := 0 ; i< len(roomUsers) ; i++{
			// emitting event to start game
			log.Println("emitting start-game event")
			socket := roomUsers[i].conn
			packet.Type="start-game"
			players := []UserClient{}
			for j:= 0 ; j < len(roomUsers) ; j++{
				players=append(players, UserClient{
					Username: roomUsers[j].username,
					UserId: roomUsers[j].userId,
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
		board := make([]interface{}, 6)
		for i := 0; i < 6; i++ {
			board[i] = make([]int, 6)
		}
		room.boardState = board
		room.startTime = time.Now().Format("2006-01-02 15:04:05")
		room.currentPlayer = 1
		rooms[roomId] = room
	}
	}
		


	
}
}

func setUpRoutes(){
	http.HandleFunc("/ws",handleConnection)
	http.HandleFunc("/heartbeat",func(w http.ResponseWriter , r *http.Request){
		w.Write([]byte("alive"))
	})

}
func StartWebSocketServer(){
	setUpRoutes()
	http.ListenAndServe(":4000",nil)
}