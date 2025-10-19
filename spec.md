
### ## High-Level System Design (Serverless)

This design is 100% event-driven, meaning components react to events (like an API request or a new message) rather than being constantly connected.

**Core System Graph:**
`Client → API Gateway → Go API Function → Message Queue → Go Worker Function → Database`



---

### ## Component Breakdown

Here are the concepts and connections for each piece of the system.

* **1. API Gateway (The "Front Door")**
    * **Concept:** This is a managed service that acts as your public-facing entry point. It replaces a traditional web server (like Nginx).
    * **Connection:** It receives the client's `POST /payments` HTTP request and knows one thing: "For this specific request, trigger the Go API Function." It handles security, request throttling, and routing for you.

* **2. Go API Function (AWS Lambda)**
    * **Concept:** This is your first Go program, compiled and uploaded as a serverless function. Its only job is validation and delegation. It is stateless and built to run in under a second.
    * **Connection:** It's triggered by the **API Gateway**. It reads the request, checks the `Idempotency-Key` (by querying the **Database**), and immediately places a "job message" onto the **Message Queue**. It then instantly returns a `202 Accepted` response.

* **3. Message Queue (AWS SQS)**
    * **Concept:** This is the asynchronous buffer that decouples your fast API from your slower payment processing. It's a simple, reliable "to-do list" for payment jobs.
    * **Connection:** It receives the job message from the **Go API Function**. It holds this message safely until a worker is ready to process it.

* **4. Go Worker Function (AWS Lambda)**
    * **Concept:** This is your second Go program, also a serverless function. This is the "heavy lifter" that contains the core orchestration logic (the "stablecoin sandwich").
    * **Connection:** It is configured to be automatically triggered by new messages on the **Message Queue**. It pulls the job, updates the **Database** status to `PROCESSING`, makes the mock on-ramp/off-ramp calls, and finally updates the **Database** record to `COMPLETED` or `FAILED`.

* **5. Database (Amazon DynamoDB)**
    * **Concept:** A serverless NoSQL database. It's chosen for its massive scale and millisecond latency, which is perfect for the fast idempotency check.
    * **Connection:** It's read from and written to by *both* Go functions. The **API Function** uses it to check for duplicate `Idempotency-Key`s, and the **Worker Function** uses it to update the payment status throughout its lifecycle.

---

### ## End-to-End System Flow

This is how a single payment request travels through the serverless system:

1.  A client sends a `POST /payments` request with a unique `Idempotency-Key` header.
2.  **API Gateway** receives the request and triggers the **Go API Function**.
3.  The **Go API Function** instantly queries **DynamoDB** for the `Idempotency-Key`.
4.  Seeing it's a new key, the function writes a "job ticket" (e.g., `{ "paymentId": "123", "amount": 1000, "currency": "EUR" }`) to the **SQS Queue**.
5.  The **Go API Function** finishes, and API Gateway returns a `202 Accepted` to the client. The client's connection ends here, less than a second after it started.
6.  *Independently*, the message in **SQS** automatically triggers the **Go Worker Function**.
7.  The **Worker Function** executes the payment logic (mock on-ramp, mock off-ramp for EUR).
8.  The **Worker** writes the final `COMPLETED` status to the **DynamoDB** record.
9.  (For the Events requirement): The **Worker** would then place one final message on another queue (or an SNS topic) to trigger a separate *Webhook Function*, which sends the status update to the client.