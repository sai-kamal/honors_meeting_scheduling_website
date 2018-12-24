function myFunction() {
    var x = document.getElementById("myTopnav");
    if (x.className === "topnav") {
        x.className += " responsive";
    } else {
        x.className = "topnav";
    }
}

function showLogInModal() {
    var x = document.getElementById('login');
    x.style.display = 'block';
}