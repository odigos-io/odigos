package dev.keyval.kvshop.frontend;

import org.springframework.web.bind.annotation.CrossOrigin;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.beans.factory.annotation.Autowired;

@RestController
public class HelloController {

    @Autowired
    public HelloController() {
        // Constructor logic can be added here if necessary
    }

    @CrossOrigin(origins = "*")
    @GetMapping("/")
    public String printHello() {
        System.out.println("Received request for Hello");
        return "Hello";
    }
}
