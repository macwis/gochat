window.addEventListener("DOMContentLoaded", (_) => {
    var prefix = "";
    hosts_local = ["127.0.0.1", "localhost"]
    if (hosts_local.some(function(v) { return window.location.host.indexOf(v) >= 0; })) {
      prefix = "ws";
    } else {
      prefix = "wss";
    }
    let websocket = new WebSocket(prefix + "://" + window.location.host + "/websocket");
    let room = document.getElementById("chat-text");
  
    websocket.addEventListener("message", function (e) {
      let data = JSON.parse(e.data);
      // creating html element
      let p = document.createElement("p");
      p.innerHTML = `<strong>${data.username}</strong>: ${data.text}`;
  
      room.append(p);
      room.scrollTop = room.scrollHeight; // Auto scroll to the bottom
    });
  
    let form = document.getElementById("input-form");
    form.addEventListener("submit", function (event) {
      event.preventDefault();
      let username = document.getElementById("input-username");
      let text = document.getElementById("input-text");
      websocket.send(
        JSON.stringify({
          username: username.value,
          text: text.value,
        })
      );
      text.value = "";
    });
  });
  