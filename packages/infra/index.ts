import { loadConfig } from './src/config';
import { deployStack } from './src/stack';

const outputs = deployStack(loadConfig());

export const webUrl = outputs.webUrl;
export const apiUrl = outputs.apiUrl;
export const secretArn = outputs.secretArn;
export const neonProjectId = outputs.neonProjectId;
export const jobsQueueUrl = outputs.jobsQueueUrl;
export const migrateFunctionName = outputs.migrateFunctionName;
