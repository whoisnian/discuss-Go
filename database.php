<?php
$dbhost = 'localhost';
$dbname = 'discuss';
$dbuser = 'username';
$dbpass = 'password';

$con = mysqli_connect($dbhost, $dbuser, $dbpass);
if(!$con) {
    die('Could not connect: '.mysqli_error());
}

mysqli_select_db($con, $dbname);

mysqli_query($con, 'create table user(
                    userid int not null auto_increment,
                    username varchar(50) not null unique,
                    password varchar(50) not null,
                    time datetime not null,
                    primary key (userid))');

mysqli_query($con, 'create table message(
                    messageid int not null auto_increment,
                    username varchar(50) not null,
                    content varchar(1500) not null,
                    time datetime not null,
                    primary key (messageid))');

date_default_timezone_set("Asia/Shanghai");
$Time = date("Y-m-d H:i:s");
mysqli_query($con, "insert into user (username, password, time) values ('guest', 'guest', '$Time')");

mysqli_close($con);
?>
