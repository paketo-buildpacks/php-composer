<?php
    require 'vendor/autoload.php';

    $log = new Monolog\Logger('my-log');
    $log->pushHandler(new Monolog\Handler\StreamHandler('app.log', Monolog\Logger::WARNING));
    $log->addWarning('some log message');

    print "OK";
?>