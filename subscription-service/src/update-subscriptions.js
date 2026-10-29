import { readdir } from 'node:fs/promises';
import path from 'path';
import process from 'process';
import { pushVacancies } from "./grpc-client.js";

const subscriptionsDirectory = path.join(import.meta.dirname, "subscriptions");
const subscriptionFiles = await readdir(subscriptionsDirectory);

const subscriptions = await Promise.allSettled(
    subscriptionFiles
        .filter((file) => file.endsWith('.js'))
        .map(
            file => import(path.join(subscriptionsDirectory, file))
        )
)

let allVacancies = []

for (const subscription of subscriptions.filter(result => result.status === "fulfilled")) {
    const  result = await subscription.value.updateSubscription()
    for (const vacancy of result) {
        allVacancies.push(vacancy);
    }
}

console.log("Pushing vacancies...")
const ack = await pushVacancies(allVacancies);
console.log(`GRPC server returned ack = ${ack}`)