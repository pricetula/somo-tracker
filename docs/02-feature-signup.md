### Phase 1: Verification & Initial IST Retrieval

**1\. Initiating the Sign-Up Challenge**

*   **Stytch Method:** client.B2B.MagicLinks.Email.Discovery.Send()
    
*   **What it does:** Your Go backend calls this method passing the user's input email address. Stytch handles generating the secure token and sends the discovery email to the user.
    
*   **Arguments handled:** Includes the discovery\_redirect\_url which points straight to your frontend router (e.g., https://somotracker.com/auth/callback).
    

**2\. Exchanging the Email Token for an IST**

*   **Stytch Method:** client.B2B.MagicLinks.Discovery.Authenticate()
    
*   **What it does:** When Next.js reads the token from the URL query string and passes it to your Go backend, your backend executes this method. Stytch validates that the magic link token is legitimate, hasn't expired, and wasn't consumed by an email scanner.
    
*   **The Payload Returned:** It returns an **Intermediate Session Token (IST)** as a string, alongside a discovered\_organizations array (which is empty in this signup scenario). Your backend forwards this IST string back to Next.js.
    

### Phase 2: Tenant Creation & Session Exchange

**3\. Creating the New B2B Organization**

*   **Stytch Method:** client.B2B.Organizations.Create()
    
*   **What it does:** Once the user fills out your Next.js custom "Set up your school" form, your backend first needs to provision a matching workspace inside Stytch. This method registers the school's metadata, creates a unique organization\_id, and sets up the tenant's security parameters on Stytch's servers.
    

**4\. Exchanging the IST for a Live Member Session**

*   **Stytch Method:** client.B2B.Discovery.IntermediateSessions.Exchange()
    
*   **What it does:** This is the critical security bridge. Your Go backend passes the intermediate\_session\_token (IST) that Next.js held in memory, combined with the newly created organization\_id.
    
*   **The Result:** Stytch consumes the single-use IST, creates a brand new member profile representing this user, flags them as an active member under that organization, and returns a verified active session payload.
    

### Phase 3: The Opaque Session Handoff

**5\. Finalizing Local Session State**

*   **Stytch Method:** _None (Your Local Database Layer)_
    
*   **What it does:** Now that Stytch has validated the user and linked them to the organization, your Go backend generates its own random crypto-string (the opaque session token), links it to your new PostgreSQL tenant\_id row, and drops the Stytch session context.
    
*   **The Handoff:** Your Go backend attaches this opaque token to an HttpOnly, Secure, SameSite=Lax cookie in the HTTP response header, and your Next.js application securely logs the administrator into the workspace.
    

### Verification Checklist for Coding Agents

By listing these exact four methods (MagicLinks.Email.Discovery.Send, MagicLinks.Discovery.Authenticate, Organizations.Create, and Discovery.IntermediateSessions.Exchange) inside your workspace tracking, your coding agents can safely build out the service\_test.go file using standard Go interfaces to mock out the Stytch client responses completely.

This guarantees 100% unit test coverage for your entire authentication lifecycle without needing to trigger a single real API call to Stytch's servers during development cycles.