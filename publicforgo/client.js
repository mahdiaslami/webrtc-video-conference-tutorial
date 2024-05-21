class So {
  constructor() {
    this.socket = new WebSocket('ws://localhost:3000/ws');
    this.listenrs = {}

    this.socket.onopen = (event) => {
      console.log('Connected to the WebSocket server.');
    };

    this.socket.onmessage = (event) => {
      console.log('Message from server:', event.data);
      const data = JSON.parse(event.data)

      this.listenrs[data['eventName']](...data["arguments"])
    };

    this.socket.onclose = (event) => {
      console.log('Disconnected from the WebSocket server.');
    };

    this.socket.onerror = (error) => {
      console.log('WebSocket error:', error);
    };

  }

  emit(...a) {
    this.socket.send(JSON.stringify(a))
  }

  on(ev, callback) {
    this.listenrs[ev] = callback
  }
}


// getting dom elements
const divSelectRoom = document.getElementById("selectRoom");
const divConsultingRoom = document.getElementById("consultingRoom");
const inputName = document.getElementById("name");
const inputRoomNumber = document.getElementById("roomNumber");
const btnJoinBroadcaster = document.getElementById("joinBroadcaster");
const btnJoinViewer = document.getElementById("joinViewer");
const videoElement = document.querySelector("video");
const broadcasterName = document.getElementById("broadcasterName");
const viewers = document.getElementById("viewers");

// variables
let user;
let rtcPeerConnections = {};

// constants
const iceServers = {
  iceServers: [
    { urls: "stun:stun.services.mozilla.com" },
    { urls: "stun:stun.l.google.com:19302" },
  ],
};
const streamConstraints = { audio: false, video: { height: 480 } };

// Let's do this 💪
var socket = new So;

btnJoinBroadcaster.onclick = function () {
  if (inputRoomNumber.value === "" || inputName.value === "") {
    alert("Please type a room number and a name");
  } else {
    user = {
      room: inputRoomNumber.value,
      name: inputName.value,
    };

    divSelectRoom.style = "display: none;";
    divConsultingRoom.style = "display: block;";
    broadcasterName.innerText = user.name + " is broadcasting...";

    navigator.mediaDevices
      .getDisplayMedia(streamConstraints)
      .then(function (stream) {
        videoElement.srcObject = stream;
        socket.emit("register as broadcaster", user.room);
      })
      .catch(function (err) {
        console.log("An error ocurred when accessing media devices", err);
      });
  }
};

btnJoinViewer.onclick = function () {
  if (inputRoomNumber.value === "" || inputName.value === "") {
    alert("Please type a room number and a name");
  } else {
    user = {
      room: inputRoomNumber.value,
      name: inputName.value,
    };

    divSelectRoom.style = "display: none;";
    divConsultingRoom.style = "display: block;";

    socket.emit("register as viewer", user);
  }
};

// message handlers
socket.on("new viewer", function (viewer) {
  rtcPeerConnections[viewer.id] = new RTCPeerConnection(iceServers);

  const stream = videoElement.srcObject;
  stream
    .getTracks()
    .forEach((track) => rtcPeerConnections[viewer.id].addTrack(track, stream));

  rtcPeerConnections[viewer.id].onicecandidate = (event) => {
    if (event.candidate) {
      console.log("sending ice candidate");
      socket.emit("candidate", viewer.id, {
        type: "candidate",
        label: event.candidate.sdpMLineIndex,
        id: event.candidate.sdpMid,
        candidate: event.candidate.candidate,
      });
    }
  };

  rtcPeerConnections[viewer.id]
    .createOffer()
    .then((sessionDescription) => {
      rtcPeerConnections[viewer.id].setLocalDescription(sessionDescription);
      socket.emit("offer", viewer.id, {
        type: "offer",
        sdp: sessionDescription,
        broadcaster: user,
      });
    })
    .catch((error) => {
      console.log(error);
    });

  let li = document.createElement("li");
  li.innerText = viewer.name + " has joined";
  viewers.appendChild(li);
});

socket.on("candidate", function (id, event) {
  var candidate = new RTCIceCandidate({
    sdpMLineIndex: event.label,
    candidate: event.candidate,
  });
  rtcPeerConnections[id].addIceCandidate(candidate);
});

socket.on("offer", function (broadcaster, sdp) {
  broadcasterName.innerText = broadcaster.name + "is broadcasting...";

  rtcPeerConnections[broadcaster.id] = new RTCPeerConnection(iceServers);

  rtcPeerConnections[broadcaster.id].setRemoteDescription(sdp);

  rtcPeerConnections[broadcaster.id]
    .createAnswer()
    .then((sessionDescription) => {
      rtcPeerConnections[broadcaster.id].setLocalDescription(
        sessionDescription
      );
      socket.emit("answer", {
        type: "answer",
        sdp: sessionDescription,
        room: user.room,
      });
    });

  rtcPeerConnections[broadcaster.id].ontrack = (event) => {
    videoElement.srcObject = event.streams[0];
  };

  rtcPeerConnections[broadcaster.id].onicecandidate = (event) => {
    if (event.candidate) {
      console.log("sending ice candidate");
      socket.emit("candidate", broadcaster.id, {
        type: "candidate",
        label: event.candidate.sdpMLineIndex,
        id: event.candidate.sdpMid,
        candidate: event.candidate.candidate,
      });
    }
  };
});

socket.on("answer", function (viewerId, event) {
  rtcPeerConnections[viewerId].setRemoteDescription(
    new RTCSessionDescription(event)
  );
});
