const DEPLOYMENT_PREFIX = "deployment-";
const STATEFUL_SET_PREFIX = "statefulset-";

// stripPrefix removes StatefulSet- and Deployment- prefixes
export function stripPrefix(name:string):string {
    if (name.startsWith(DEPLOYMENT_PREFIX)){
        return name.substring(DEPLOYMENT_PREFIX.length)
    } else if (name.startsWith(STATEFUL_SET_PREFIX)){
        return name.substring(STATEFUL_SET_PREFIX.length)
    }

    return name
}