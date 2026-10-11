package com.example.headsampling;

import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.LinkedHashMap;
import java.util.Map;

@RestController
@ConditionalOnProperty(name = "http.match.enable-routes", havingValue = "true", matchIfMissing = true)
public class HttpMatchController {

    @GetMapping("/http-match/control/no-rule")
    public Map<String, Object> httpMatchControl() {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("category", "control");
        m.put("endpoint", "/http-match/control/no-rule");
        m.put("description", "no sampling rule targets this path; traces follow default pipeline behavior");
        return m;
    }

    @GetMapping("/http-match/exact/target")
    public Map<String, Object> httpMatchExactTarget() {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("category", "exact_route");
        m.put("endpoint", "/http-match/exact/target");
        m.put("description", "exact HTTP route match (full literal path)");
        return m;
    }

    @PostMapping("/http-match/exact/post-target")
    public Map<String, Object> httpMatchExactPost() {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("category", "exact_route_post");
        m.put("endpoint", "/http-match/exact/post-target");
        m.put("description", "POST-only exact path for client outbound tests");
        return m;
    }

    @GetMapping("/http-match/prefix/segment")
    public Map<String, Object> httpMatchPrefixSegment() {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("category", "prefix_route");
        m.put("endpoint", "/http-match/prefix/segment");
        m.put("description", "prefix HTTP route match (first segment after /http-match/prefix)");
        return m;
    }

    @GetMapping("/http-match/prefix/segment/nested")
    public Map<String, Object> httpMatchPrefixNested() {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("category", "prefix_route");
        m.put("endpoint", "/http-match/prefix/segment/nested");
        m.put("description", "prefix HTTP route match (deeper path under the same static prefix)");
        return m;
    }

    @GetMapping("/http-match/texact/{resourceId}")
    public Map<String, Object> httpMatchTexact(@PathVariable String resourceId) {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("category", "templatized_exact");
        m.put("endpoint", "/http-match/texact/:resourceId");
        m.put("resource_id", resourceId);
        m.put("description", "templatized exact route (one dynamic segment)");
        return m;
    }

    @GetMapping("/http-match/tprefix/{tenantId}/items")
    public Map<String, Object> httpMatchTprefixItems(@PathVariable String tenantId) {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("category", "templatized_prefix");
        m.put("endpoint", "/http-match/tprefix/:tenantId/items");
        m.put("tenant_id", tenantId);
        m.put("description", "templatized prefix (tenant/items)");
        return m;
    }

    @GetMapping("/http-match/tprefix/{tenantId}/items/{itemId}")
    public Map<String, Object> httpMatchTprefixItem(
            @PathVariable String tenantId, @PathVariable String itemId) {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("category", "templatized_prefix");
        m.put("endpoint", "/http-match/tprefix/:tenantId/items/:itemId");
        m.put("tenant_id", tenantId);
        m.put("item_id", itemId);
        m.put("description", "templatized prefix with extra path under items");
        return m;
    }
}
