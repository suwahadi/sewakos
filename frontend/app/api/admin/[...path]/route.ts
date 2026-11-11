import { createScopedProxy } from "@/lib/api/scoped-proxy";

const forward = createScopedProxy("/admin");

export { forward as DELETE, forward as GET, forward as PATCH, forward as POST, forward as PUT };
