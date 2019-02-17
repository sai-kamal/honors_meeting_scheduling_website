document.addEventListener('DOMContentLoaded', function () {
     var elems = document.querySelectorAll('.modal');
     var instances = M.Modal.init(elems);
});

function see_meeting(meeting_name, timeSpace, timeDiff) {
    $('#meeting_name').val(meeting_name);
    var elem = document.querySelector('#joinMeeting');
    var instance = M.Modal.getInstance(elem);
    var selectDelay = $('#delay');
    var temp = Math.floor(timeSpace/timeDiff)+1;
    for (let i = 0; i < temp; i++) {
        var opt = $("<option>", {value: i, text: String(i*timeDiff) + " min"})
        selectDelay.append(opt);
    }
    $('select').formSelect(); //materialize css
    instance.open();
}

function see_log_meeting(meeting_name) {
    window.location.pathname = '../seeLog' + meeting_name;
}