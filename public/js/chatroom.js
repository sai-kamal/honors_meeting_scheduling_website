//global variables
var message, message, userName, output, socket, hostName;

$(document).ready(function(){
    message = document.getElementById("message")
    userName = document.getElementById("userName")
    output = document.getElementById("output")
    hostName = location.hostname;
    if(hostName == "localhost") {
        socket = new WebSocket("ws://localhost:3000/chat")
    }

    socket.onopen = function () {
        $("#temp").text("Status: Connected");
    }

    socket.onmessage = function (e) {
        var messageDetails = JSON.parse(e.data);
        console.log(messageDetails)
        var divNode = document.createElement("div");
        var userSpanNode = document.createElement("span");
        var boldNode = document.createElement("strong");
        var messageSpanNode = document.createElement("span");
        userSpanNode.setAttribute("class", "col col-md-4");
        messageSpanNode.setAttribute("class", "col col-md-8");
        divNode.setAttribute("class", "row");
        var userTextnode = document.createTextNode(messageDetails.username + "-");
        var messageTextnode = document.createTextNode(messageDetails.message + "\n");
        boldNode.appendChild(userTextnode);
        userSpanNode.appendChild(boldNode);
        messageSpanNode.appendChild(messageTextnode);
        divNode.appendChild(userSpanNode);
        divNode.appendChild(messageSpanNode);
        document.getElementById("output").appendChild(divNode);
    }
});

function send() {
    var time = new Date();
    var currenTimeStamp = time.toLocaleString('en-US', {
        hour: 'numeric',
        minute: 'numeric',
        hour12: true
    });
    var messageDetails = {
        username: userName.value,
        message: message.value,
        timestamp: currenTimeStamp
    }
    socket.send(JSON.stringify(messageDetails));
    message.value = "";
}

(function (i, s, o, g, r, a, m) {
    i['GoogleAnalyticsObject'] = r;
    i[r] = i[r] || function () {
        (i[r].q = i[r].q || []).push(arguments)
    }, i[r].l = 1 * new Date();
    a = s.createElement(o),
        m = s.getElementsByTagName(o)[0];
    a.async = 1;
    a.src = g;
    m.parentNode.insertBefore(a, m)
})(window, document, 'script', 'https://www.google-analytics.com/analytics.js', 'ga');

ga('create', 'UA-105302950-1', 'auto');
ga('send', 'pageview');