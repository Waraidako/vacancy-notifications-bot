import grpc from "@grpc/grpc-js"
import protoLoader from "@grpc/proto-loader"
import path from 'path';
import process from 'process';

const protoPath = path.join(import.meta.dirname, "../../proto/subscriptions.proto")

const packageDefinition = protoLoader.loadSync(protoPath);
const proto = grpc.loadPackageDefinition(packageDefinition);

const client = new proto.vacancies.Subscriptions(
    `bot:${"BOT_GRPC_PORT" in process.env ? process.env.BOT_GRPC_PORT : 50051}`,
    grpc.credentials.createInsecure()
)

export async function pushVacancies(vacancies) {
    return new Promise((resolve, reject) => {
        client.pushVacancies({ vacancies }, (error, response) => {
            if (error) {
                reject(error);
                return;
            }
            resolve(response.ack)
        })
    })
}