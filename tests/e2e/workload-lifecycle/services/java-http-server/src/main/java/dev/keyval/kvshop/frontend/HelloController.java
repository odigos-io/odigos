package dev.keyval.kvshop.frontend;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.*;

@RestController
public class HelloController {

    @Autowired
    public HelloController() {}

@CrossOrigin(origins = "*")
@GetMapping("/")
public String printHello() {
   return "Hello";
}
}
