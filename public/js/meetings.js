document.addEventListener('DOMContentLoaded', function () {
     var elems = document.querySelectorAll('.modal');
     var instances = M.Modal.init(elems);
});

function see_meeting(meetingName, timeSpace, timeDiff, origExpect) {
    $('#meeting_name').val(meetingName);
    var elem = document.querySelector('#joinMeeting');
    var instance = M.Modal.getInstance(elem);
    var selectDelay = $('#delay');
    var temp = Math.floor((timeSpace - (origExpect * timeDiff)) / timeDiff);
    for (let i = 0; i <= temp; i++) {
        var opt = $("<option>", {value: origExpect+i, text: String(i*timeDiff) + " min"})
        selectDelay.append(opt);
    }
    $('select').formSelect(); //materialize css
    instance.open();
}

function see_log_meeting(meetingName) {
    window.location.pathname = '../seeLog' + meetingName;
}