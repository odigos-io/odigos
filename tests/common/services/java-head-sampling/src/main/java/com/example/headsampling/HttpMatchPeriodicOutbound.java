package com.example.headsampling;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClientException;
import org.springframework.web.client.RestTemplate;

import javax.annotation.PostConstruct;
import javax.annotation.PreDestroy;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

@Component
public class HttpMatchPeriodicOutbound {

    private static final Logger log = LoggerFactory.getLogger(HttpMatchPeriodicOutbound.class);
    private static final int PORT = 8080;

    private final RestTemplate restTemplate;

    private ScheduledExecutorService scheduler;

    public HttpMatchPeriodicOutbound(RestTemplate restTemplate) {
        this.restTemplate = restTemplate;
    }

    @PostConstruct
    public void start() {
        if (!"true".equalsIgnoreCase(env("HTTP_MATCH_PERIODIC_OUTBOUND", "false"))) {
            return;
        }
        String defaultPeerBase = stripTrailingSlash(env("HTTP_MATCH_PEER_BASE_URL", "http://127.0.0.1:" + PORT));
        String exactPeerBase = stripTrailingSlash(env("HTTP_MATCH_EXACT_PEER_BASE_URL", defaultPeerBase));
        long intervalMs = parseIntervalMs(env("HTTP_MATCH_OUTBOUND_INTERVAL_MS", "10000"));
        String exactGetPath = "/http-match/exact/target";
        String postExactPath = "/http-match/exact/post-target";
        String[] prefixAndTemplatedPaths = {
                "/http-match/prefix/segment",
                "/http-match/prefix/segment/nested",
                "/http-match/texact/out-peer-res",
                "/http-match/tprefix/out-peer-tenant/items",
                "/http-match/tprefix/out-peer-tenant/items/out-peer-item",
        };
        int requestsPerTick = 2 + 1 + prefixAndTemplatedPaths.length;

        Runnable fireOutbound = () -> {
            getIgnoreErrors(defaultPeerBase + exactGetPath);
            getIgnoreErrors(exactPeerBase + exactGetPath);
            postIgnoreErrors(defaultPeerBase + postExactPath);
            for (String relPath : prefixAndTemplatedPaths) {
                getIgnoreErrors(defaultPeerBase + relPath);
            }
        };

        log.info(
                "http-match periodic outbound every {}ms → {} requests/tick (GET exact ×2 + POST exact + prefix/templated GETs; bases {} / {})",
                intervalMs,
                requestsPerTick,
                defaultPeerBase,
                exactPeerBase);

        scheduler = Executors.newSingleThreadScheduledExecutor(r -> {
            Thread t = new Thread(r, "http-match-outbound");
            t.setDaemon(false);
            return t;
        });
        scheduler.scheduleAtFixedRate(fireOutbound, intervalMs, intervalMs, TimeUnit.MILLISECONDS);
    }

    @PreDestroy
    public void stop() {
        if (scheduler != null) {
            scheduler.shutdown();
            try {
                if (!scheduler.awaitTermination(10, TimeUnit.SECONDS)) {
                    scheduler.shutdownNow();
                }
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                scheduler.shutdownNow();
            }
        }
    }

    private void getIgnoreErrors(String url) {
        try {
            restTemplate.getForEntity(url, String.class);
        } catch (RestClientException e) {
            log.error("http-match periodic outbound failed {} — {}", url, e.getMessage());
        }
    }

    private void postIgnoreErrors(String url) {
        try {
            restTemplate.postForEntity(url, null, String.class);
        } catch (RestClientException e) {
            log.error("http-match periodic outbound POST failed {} — {}", url, e.getMessage());
        }
    }

    private static String env(String key, String defaultValue) {
        String v = System.getenv(key);
        return v != null && !v.isEmpty() ? v : defaultValue;
    }

    private static String stripTrailingSlash(String base) {
        if (base.endsWith("/")) {
            return base.substring(0, base.length() - 1);
        }
        return base;
    }

    private static long parseIntervalMs(String raw) {
        try {
            long n = Long.parseLong(raw);
            if (n < 1000) {
                return 10000;
            }
            return n;
        } catch (NumberFormatException e) {
            return 10000;
        }
    }
}
