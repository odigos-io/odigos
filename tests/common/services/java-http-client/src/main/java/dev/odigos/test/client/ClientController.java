package dev.odigos.test.client;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;

@RestController
public class ClientController {

    private final WebClient webClient;

    public ClientController(@Value("${target.url:http://java-supported-version:3000}") String targetUrl) {
        this.webClient = WebClient.builder().baseUrl(targetUrl).build();
    }

    @GetMapping("/call-via-webclient")
    public Mono<String> callViaWebClient() {
        return webClient.get()
                .uri("/")
                .retrieve()
                .bodyToMono(String.class)
                .map(body -> "WebClient received: " + body);
    }

    @GetMapping("/health")
    public Mono<String> health() {
        return Mono.just("OK");
    }
}
