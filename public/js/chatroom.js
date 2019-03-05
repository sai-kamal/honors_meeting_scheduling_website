//global variables
var message, message, userName, output, socket, hostName;

function delayPopUp(timeSpace, timeDiff, origExpect) {//initializes data into list everytime for user to change delay at any time
    var time = $('#timeDisplay').text();
    if (time === "Not Yet Started") {
        time = 0;
    } else {
        time = parseInt(time.split(" ")[0])/timeDiff;
    }
    var elem = document.querySelector('#delayID');
    var instance = M.Modal.init(elem);
    var selectDelay = $('#delayInChat');
    selectDelay.empty();
    var temp = Math.floor((timeSpace - (origExpect * timeDiff)) / timeDiff);
    for (let i = 0; i <= temp; i++) {
        if (origExpect + i >= time) { //equal to means that he has arrived at the meeting
            var opt = $("<option>", {
                value: origExpect + i,
                text: String(i * timeDiff) + " min"
            })
            selectDelay.append(opt);
        }
    }
    $('#delayInChat').formSelect(); //materialize css
    instance.open();
}

function changeDelay() { //sends msg to server to change delay of user
    var messageDetails = {
        type: "change_user_delay",
        time: $("#delayInChat").val(),
    }
    socket.send(JSON.stringify(messageDetails));
    var elem = document.querySelector('#delayID');
    var instance = M.Modal.init(elem);
    instance.close();
}

function sendActionToServer() {//send user action reply to server when agent asks action as reply for transfer_control
    var elem = document.querySelector('#transferControlModal');
    var instance = M.Modal.init(elem);
    instance.close();
    var messageDetails = {
        type: "transfer_control_reply",
        action: $("#actions").val(),
        action_name: $("#actions option:selected").text()
    }
    socket.send(JSON.stringify(messageDetails));

}

function openTransferControlPopUp(messageDetails) {//initializes transfer_control pop up with values for the user
    console.log(messageDetails)
    var elem = document.querySelector('#transferControlModal');
    var instance = M.Modal.init(elem);
    var selectDelay = $('#actions');
    selectDelay.empty();
    for (let i = 0; i < messageDetails.actions.length; i++) {
        // should do this according with time
        var opt = $("<option>", {
            value: messageDetails.actions[i],
            text: messageDetails.actions_names[i]
        })
        selectDelay.append(opt);
    }
    $('#actions').formSelect(); //materialize css
    instance.open();
    //close instance after some time
}

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
        if(messageDetails.type === "change_time") {
            $("#timeDisplay").text(messageDetails.time);
        } else if (messageDetails.type === "change_orig_expect") {
            $("#origExpectDisp").text(messageDetails.time);
        } else if (messageDetails.type === "change_curr_expect") {
            $("#currExpectDisp").text(messageDetails.time);
        } else if (messageDetails.type === "change_agent_expect") {
            $("#agentExpectDisp").text(messageDetails.time);
        }
        if(messageDetails.type === "transfer_control") {
            openTransferControlPopUp(messageDetails)
        }

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
    // var time = new Date();
    // var currenTimeStamp = time.toLocaleString('en-US', {
    //     hour: 'numeric',
    //     minute: 'numeric',
    //     hour12: true
    // });
    var messageDetails = {
        username: userName.value,
        message: message.value,
        // timestamp: currenTimeStamp
    }
    socket.send(JSON.stringify(messageDetails));
    message.value = "";
}

// (function (i, s, o, g, r, a, m) {
//     i['GoogleAnalyticsObject'] = r;
//     i[r] = i[r] || function () {
//         (i[r].q = i[r].q || []).push(arguments)
//     }, i[r].l = 1 * new Date();
//     a = s.createElement(o),
//         m = s.getElementsByTagName(o)[0];
//     a.async = 1;
//     a.src = g;
//     m.parentNode.insertBefore(a, m)
// })(window, document, 'script', 'https://www.google-analytics.com/analytics.js', 'ga');

// ga('create', 'UA-105302950-1', 'auto');
// ga('send', 'pageview');