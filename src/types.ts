import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface MongoDBQuery extends DataQuery {
  database: string;
  collection: string;
  timestampField: string;
  timestampFormat: string;
  labelFields: string[];
  legendFormat: string;
  valueFields: string[];
  valueFieldTypes: string[];
  aggregation: string;
  autoTimeBound: boolean;
  autoTimeBoundAtStart: boolean;
  autoTimeSort: boolean;
  schemaInference: boolean;
  schemaInferenceDepth: number;
}

export enum MongoDBQueryType {
    Timeseries = "Timeseries",
    Table = "Table",
};

export const defaultQuery: Partial<MongoDBQuery> = {
    database: "my_db",
    collection: "my_collection",
    queryType: MongoDBQueryType.Timeseries,
    timestampField: "timestamp",
    timestampFormat: "",
    labelFields: [ "sensorID" ],
    legendFormat: "",
    valueFields: [ "measurement" ],
    valueFieldTypes: [ "float64" ],
    aggregation: JSON.stringify([
        { 
            "$project": { 
                "timestamp": 1, 
                "sensorID": "$metadata.sensorID",
                "measurement": 1, 
                "_id": 0 
            }
        }
    ]),
    autoTimeBound: false,
    autoTimeSort: false,
    schemaInference: false,
    schemaInferenceDepth: 20,
};

export interface MongoDBVariableQuery {
    database: string;
    collection: string;
    aggregation: string;
    fieldName: string;
    fieldType: string;
};

export const defaultVariableQuery: Partial<MongoDBVariableQuery> = {
    database: "my_db",
    collection: "my_collection",
    aggregation: JSON.stringify([
        {"$group":{"_id":"$label", "count": {"$sum":1}}}
    ]),
    fieldName: "_id",
    fieldType: "string"
};

/**
 * These are options configured for each DataSource instance.
 */
export interface MongoDBDataSourceOptions extends DataSourceJsonData {
  url?: string;
  tls?: boolean;
  tlsInsecure?: boolean;
  tlsCertificate?: string;
  tlsCa?: string;
  tlsServerName?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MongoDBSecureJsonData {
    username?: string;
    password?: string;
    tlsCertificateKey?: string;
}
