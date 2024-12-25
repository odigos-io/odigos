# Development Guide

This guide provides advanced instructions for contributors and maintainers, covering topics such as debugging specific components, analyzing performance profiles, and working with internal tools. It complements the `CONTRIBUTING.md` by offering insights into advanced development workflows and optimization techniques.

---

## CPU and Memory Profiling for the Collectors

### Step 1: Port Forward the Gateway or Data Collection Pod
Forward the relevant pod to your local machine to enable profiling access:

kubectl port-forward pod/<pod-name> -n odigos-system 1777:1777


### Step 2: Collect Profiling Data

- **CPU Profile**  
   Captures data about the amount of time your application spends executing functions. Use this profile to identify performance bottlenecks, optimize CPU-intensive operations, and analyze which parts of the code consume the most CPU resources.  

   ``` bash
    curl -o cpu_profile.prof http://localhost:1777/debug/pprof/profile?seconds=30 
   ```

- **Heap Memory Profile**  
   Captures a snapshot of memory currently in use by your application after the latest garbage collection. Use this profile to identify memory leaks, track high memory usage, and analyze memory consumption by specific parts of the code.  
   ``` bash
   curl -o heap.out http://localhost:1777/debug/pprof/heap
   ```

- **Historical Memory Allocation**  
   Provides insights into all memory allocations made by the program since it started running, including memory that has already been freed by the garbage collector (GC). This is useful for understanding memory allocation patterns and optimizing allocation behavior.  
    ``` bash
   curl -o allocs.out http://localhost:1777/debug/pprof/allocs
   ```

### Step 3: Analyze the Profiles
After collecting the profiling data, use the `go tool pprof` command to analyze the profiles visually in your web browser. Replace `<output file>` with the appropriate file (`cpu_profile.prof`, `heap.out`, or `allocs.out`):
``` bash
go tool pprof -http=:8080 <output file>
```
This opens an interactive interface in your browser where you can:
- **Visualize Hotspots**: View flame graphs or directed graphs for easy identification of bottlenecks.
- **Drill Down**: Explore specific functions or memory allocations for detailed insights.