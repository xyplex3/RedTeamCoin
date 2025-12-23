package mining;

import static io.grpc.MethodDescriptor.generateFullMethodName;

/**
 * <pre>
 * Mining service for pool communication
 * </pre>
 */
@javax.annotation.Generated(
    value = "by gRPC proto compiler (version 1.60.0)",
    comments = "Source: mining.proto")
@io.grpc.stub.annotations.GrpcGenerated
public final class MiningPoolGrpc {

  private MiningPoolGrpc() {}

  public static final java.lang.String SERVICE_NAME = "mining.MiningPool";

  // Static method descriptors that strictly reflect the proto.
  private static volatile io.grpc.MethodDescriptor<mining.Mining.MinerInfo,
      mining.Mining.RegistrationResponse> getRegisterMinerMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "RegisterMiner",
      requestType = mining.Mining.MinerInfo.class,
      responseType = mining.Mining.RegistrationResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<mining.Mining.MinerInfo,
      mining.Mining.RegistrationResponse> getRegisterMinerMethod() {
    io.grpc.MethodDescriptor<mining.Mining.MinerInfo, mining.Mining.RegistrationResponse> getRegisterMinerMethod;
    if ((getRegisterMinerMethod = MiningPoolGrpc.getRegisterMinerMethod) == null) {
      synchronized (MiningPoolGrpc.class) {
        if ((getRegisterMinerMethod = MiningPoolGrpc.getRegisterMinerMethod) == null) {
          MiningPoolGrpc.getRegisterMinerMethod = getRegisterMinerMethod =
              io.grpc.MethodDescriptor.<mining.Mining.MinerInfo, mining.Mining.RegistrationResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "RegisterMiner"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  mining.Mining.MinerInfo.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  mining.Mining.RegistrationResponse.getDefaultInstance()))
              .setSchemaDescriptor(new MiningPoolMethodDescriptorSupplier("RegisterMiner"))
              .build();
        }
      }
    }
    return getRegisterMinerMethod;
  }

  private static volatile io.grpc.MethodDescriptor<mining.Mining.WorkRequest,
      mining.Mining.WorkResponse> getGetWorkMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "GetWork",
      requestType = mining.Mining.WorkRequest.class,
      responseType = mining.Mining.WorkResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<mining.Mining.WorkRequest,
      mining.Mining.WorkResponse> getGetWorkMethod() {
    io.grpc.MethodDescriptor<mining.Mining.WorkRequest, mining.Mining.WorkResponse> getGetWorkMethod;
    if ((getGetWorkMethod = MiningPoolGrpc.getGetWorkMethod) == null) {
      synchronized (MiningPoolGrpc.class) {
        if ((getGetWorkMethod = MiningPoolGrpc.getGetWorkMethod) == null) {
          MiningPoolGrpc.getGetWorkMethod = getGetWorkMethod =
              io.grpc.MethodDescriptor.<mining.Mining.WorkRequest, mining.Mining.WorkResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "GetWork"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  mining.Mining.WorkRequest.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  mining.Mining.WorkResponse.getDefaultInstance()))
              .setSchemaDescriptor(new MiningPoolMethodDescriptorSupplier("GetWork"))
              .build();
        }
      }
    }
    return getGetWorkMethod;
  }

  private static volatile io.grpc.MethodDescriptor<mining.Mining.WorkSubmission,
      mining.Mining.SubmissionResponse> getSubmitWorkMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "SubmitWork",
      requestType = mining.Mining.WorkSubmission.class,
      responseType = mining.Mining.SubmissionResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<mining.Mining.WorkSubmission,
      mining.Mining.SubmissionResponse> getSubmitWorkMethod() {
    io.grpc.MethodDescriptor<mining.Mining.WorkSubmission, mining.Mining.SubmissionResponse> getSubmitWorkMethod;
    if ((getSubmitWorkMethod = MiningPoolGrpc.getSubmitWorkMethod) == null) {
      synchronized (MiningPoolGrpc.class) {
        if ((getSubmitWorkMethod = MiningPoolGrpc.getSubmitWorkMethod) == null) {
          MiningPoolGrpc.getSubmitWorkMethod = getSubmitWorkMethod =
              io.grpc.MethodDescriptor.<mining.Mining.WorkSubmission, mining.Mining.SubmissionResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "SubmitWork"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  mining.Mining.WorkSubmission.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  mining.Mining.SubmissionResponse.getDefaultInstance()))
              .setSchemaDescriptor(new MiningPoolMethodDescriptorSupplier("SubmitWork"))
              .build();
        }
      }
    }
    return getSubmitWorkMethod;
  }

  private static volatile io.grpc.MethodDescriptor<mining.Mining.MinerStatus,
      mining.Mining.HeartbeatResponse> getHeartbeatMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "Heartbeat",
      requestType = mining.Mining.MinerStatus.class,
      responseType = mining.Mining.HeartbeatResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<mining.Mining.MinerStatus,
      mining.Mining.HeartbeatResponse> getHeartbeatMethod() {
    io.grpc.MethodDescriptor<mining.Mining.MinerStatus, mining.Mining.HeartbeatResponse> getHeartbeatMethod;
    if ((getHeartbeatMethod = MiningPoolGrpc.getHeartbeatMethod) == null) {
      synchronized (MiningPoolGrpc.class) {
        if ((getHeartbeatMethod = MiningPoolGrpc.getHeartbeatMethod) == null) {
          MiningPoolGrpc.getHeartbeatMethod = getHeartbeatMethod =
              io.grpc.MethodDescriptor.<mining.Mining.MinerStatus, mining.Mining.HeartbeatResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "Heartbeat"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  mining.Mining.MinerStatus.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  mining.Mining.HeartbeatResponse.getDefaultInstance()))
              .setSchemaDescriptor(new MiningPoolMethodDescriptorSupplier("Heartbeat"))
              .build();
        }
      }
    }
    return getHeartbeatMethod;
  }

  private static volatile io.grpc.MethodDescriptor<mining.Mining.MinerInfo,
      mining.Mining.StopResponse> getStopMiningMethod;

  @io.grpc.stub.annotations.RpcMethod(
      fullMethodName = SERVICE_NAME + '/' + "StopMining",
      requestType = mining.Mining.MinerInfo.class,
      responseType = mining.Mining.StopResponse.class,
      methodType = io.grpc.MethodDescriptor.MethodType.UNARY)
  public static io.grpc.MethodDescriptor<mining.Mining.MinerInfo,
      mining.Mining.StopResponse> getStopMiningMethod() {
    io.grpc.MethodDescriptor<mining.Mining.MinerInfo, mining.Mining.StopResponse> getStopMiningMethod;
    if ((getStopMiningMethod = MiningPoolGrpc.getStopMiningMethod) == null) {
      synchronized (MiningPoolGrpc.class) {
        if ((getStopMiningMethod = MiningPoolGrpc.getStopMiningMethod) == null) {
          MiningPoolGrpc.getStopMiningMethod = getStopMiningMethod =
              io.grpc.MethodDescriptor.<mining.Mining.MinerInfo, mining.Mining.StopResponse>newBuilder()
              .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
              .setFullMethodName(generateFullMethodName(SERVICE_NAME, "StopMining"))
              .setSampledToLocalTracing(true)
              .setRequestMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  mining.Mining.MinerInfo.getDefaultInstance()))
              .setResponseMarshaller(io.grpc.protobuf.ProtoUtils.marshaller(
                  mining.Mining.StopResponse.getDefaultInstance()))
              .setSchemaDescriptor(new MiningPoolMethodDescriptorSupplier("StopMining"))
              .build();
        }
      }
    }
    return getStopMiningMethod;
  }

  /**
   * Creates a new async stub that supports all call types for the service
   */
  public static MiningPoolStub newStub(io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<MiningPoolStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<MiningPoolStub>() {
        @java.lang.Override
        public MiningPoolStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new MiningPoolStub(channel, callOptions);
        }
      };
    return MiningPoolStub.newStub(factory, channel);
  }

  /**
   * Creates a new blocking-style stub that supports unary and streaming output calls on the service
   */
  public static MiningPoolBlockingStub newBlockingStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<MiningPoolBlockingStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<MiningPoolBlockingStub>() {
        @java.lang.Override
        public MiningPoolBlockingStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new MiningPoolBlockingStub(channel, callOptions);
        }
      };
    return MiningPoolBlockingStub.newStub(factory, channel);
  }

  /**
   * Creates a new ListenableFuture-style stub that supports unary calls on the service
   */
  public static MiningPoolFutureStub newFutureStub(
      io.grpc.Channel channel) {
    io.grpc.stub.AbstractStub.StubFactory<MiningPoolFutureStub> factory =
      new io.grpc.stub.AbstractStub.StubFactory<MiningPoolFutureStub>() {
        @java.lang.Override
        public MiningPoolFutureStub newStub(io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
          return new MiningPoolFutureStub(channel, callOptions);
        }
      };
    return MiningPoolFutureStub.newStub(factory, channel);
  }

  /**
   * <pre>
   * Mining service for pool communication
   * </pre>
   */
  public interface AsyncService {

    /**
     * <pre>
     * Register a new miner
     * </pre>
     */
    default void registerMiner(mining.Mining.MinerInfo request,
        io.grpc.stub.StreamObserver<mining.Mining.RegistrationResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getRegisterMinerMethod(), responseObserver);
    }

    /**
     * <pre>
     * Get mining work from pool
     * </pre>
     */
    default void getWork(mining.Mining.WorkRequest request,
        io.grpc.stub.StreamObserver<mining.Mining.WorkResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getGetWorkMethod(), responseObserver);
    }

    /**
     * <pre>
     * Submit mined block
     * </pre>
     */
    default void submitWork(mining.Mining.WorkSubmission request,
        io.grpc.stub.StreamObserver<mining.Mining.SubmissionResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getSubmitWorkMethod(), responseObserver);
    }

    /**
     * <pre>
     * Heartbeat to keep miner active
     * </pre>
     */
    default void heartbeat(mining.Mining.MinerStatus request,
        io.grpc.stub.StreamObserver<mining.Mining.HeartbeatResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getHeartbeatMethod(), responseObserver);
    }

    /**
     * <pre>
     * Stop mining
     * </pre>
     */
    default void stopMining(mining.Mining.MinerInfo request,
        io.grpc.stub.StreamObserver<mining.Mining.StopResponse> responseObserver) {
      io.grpc.stub.ServerCalls.asyncUnimplementedUnaryCall(getStopMiningMethod(), responseObserver);
    }
  }

  /**
   * Base class for the server implementation of the service MiningPool.
   * <pre>
   * Mining service for pool communication
   * </pre>
   */
  public static abstract class MiningPoolImplBase
      implements io.grpc.BindableService, AsyncService {

    @java.lang.Override public final io.grpc.ServerServiceDefinition bindService() {
      return MiningPoolGrpc.bindService(this);
    }
  }

  /**
   * A stub to allow clients to do asynchronous rpc calls to service MiningPool.
   * <pre>
   * Mining service for pool communication
   * </pre>
   */
  public static final class MiningPoolStub
      extends io.grpc.stub.AbstractAsyncStub<MiningPoolStub> {
    private MiningPoolStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected MiningPoolStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new MiningPoolStub(channel, callOptions);
    }

    /**
     * <pre>
     * Register a new miner
     * </pre>
     */
    public void registerMiner(mining.Mining.MinerInfo request,
        io.grpc.stub.StreamObserver<mining.Mining.RegistrationResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getRegisterMinerMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * Get mining work from pool
     * </pre>
     */
    public void getWork(mining.Mining.WorkRequest request,
        io.grpc.stub.StreamObserver<mining.Mining.WorkResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getGetWorkMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * Submit mined block
     * </pre>
     */
    public void submitWork(mining.Mining.WorkSubmission request,
        io.grpc.stub.StreamObserver<mining.Mining.SubmissionResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getSubmitWorkMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * Heartbeat to keep miner active
     * </pre>
     */
    public void heartbeat(mining.Mining.MinerStatus request,
        io.grpc.stub.StreamObserver<mining.Mining.HeartbeatResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getHeartbeatMethod(), getCallOptions()), request, responseObserver);
    }

    /**
     * <pre>
     * Stop mining
     * </pre>
     */
    public void stopMining(mining.Mining.MinerInfo request,
        io.grpc.stub.StreamObserver<mining.Mining.StopResponse> responseObserver) {
      io.grpc.stub.ClientCalls.asyncUnaryCall(
          getChannel().newCall(getStopMiningMethod(), getCallOptions()), request, responseObserver);
    }
  }

  /**
   * A stub to allow clients to do synchronous rpc calls to service MiningPool.
   * <pre>
   * Mining service for pool communication
   * </pre>
   */
  public static final class MiningPoolBlockingStub
      extends io.grpc.stub.AbstractBlockingStub<MiningPoolBlockingStub> {
    private MiningPoolBlockingStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected MiningPoolBlockingStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new MiningPoolBlockingStub(channel, callOptions);
    }

    /**
     * <pre>
     * Register a new miner
     * </pre>
     */
    public mining.Mining.RegistrationResponse registerMiner(mining.Mining.MinerInfo request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getRegisterMinerMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Get mining work from pool
     * </pre>
     */
    public mining.Mining.WorkResponse getWork(mining.Mining.WorkRequest request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getGetWorkMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Submit mined block
     * </pre>
     */
    public mining.Mining.SubmissionResponse submitWork(mining.Mining.WorkSubmission request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getSubmitWorkMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Heartbeat to keep miner active
     * </pre>
     */
    public mining.Mining.HeartbeatResponse heartbeat(mining.Mining.MinerStatus request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getHeartbeatMethod(), getCallOptions(), request);
    }

    /**
     * <pre>
     * Stop mining
     * </pre>
     */
    public mining.Mining.StopResponse stopMining(mining.Mining.MinerInfo request) {
      return io.grpc.stub.ClientCalls.blockingUnaryCall(
          getChannel(), getStopMiningMethod(), getCallOptions(), request);
    }
  }

  /**
   * A stub to allow clients to do ListenableFuture-style rpc calls to service MiningPool.
   * <pre>
   * Mining service for pool communication
   * </pre>
   */
  public static final class MiningPoolFutureStub
      extends io.grpc.stub.AbstractFutureStub<MiningPoolFutureStub> {
    private MiningPoolFutureStub(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      super(channel, callOptions);
    }

    @java.lang.Override
    protected MiningPoolFutureStub build(
        io.grpc.Channel channel, io.grpc.CallOptions callOptions) {
      return new MiningPoolFutureStub(channel, callOptions);
    }

    /**
     * <pre>
     * Register a new miner
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<mining.Mining.RegistrationResponse> registerMiner(
        mining.Mining.MinerInfo request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getRegisterMinerMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * Get mining work from pool
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<mining.Mining.WorkResponse> getWork(
        mining.Mining.WorkRequest request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getGetWorkMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * Submit mined block
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<mining.Mining.SubmissionResponse> submitWork(
        mining.Mining.WorkSubmission request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getSubmitWorkMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * Heartbeat to keep miner active
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<mining.Mining.HeartbeatResponse> heartbeat(
        mining.Mining.MinerStatus request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getHeartbeatMethod(), getCallOptions()), request);
    }

    /**
     * <pre>
     * Stop mining
     * </pre>
     */
    public com.google.common.util.concurrent.ListenableFuture<mining.Mining.StopResponse> stopMining(
        mining.Mining.MinerInfo request) {
      return io.grpc.stub.ClientCalls.futureUnaryCall(
          getChannel().newCall(getStopMiningMethod(), getCallOptions()), request);
    }
  }

  private static final int METHODID_REGISTER_MINER = 0;
  private static final int METHODID_GET_WORK = 1;
  private static final int METHODID_SUBMIT_WORK = 2;
  private static final int METHODID_HEARTBEAT = 3;
  private static final int METHODID_STOP_MINING = 4;

  private static final class MethodHandlers<Req, Resp> implements
      io.grpc.stub.ServerCalls.UnaryMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ServerStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.ClientStreamingMethod<Req, Resp>,
      io.grpc.stub.ServerCalls.BidiStreamingMethod<Req, Resp> {
    private final AsyncService serviceImpl;
    private final int methodId;

    MethodHandlers(AsyncService serviceImpl, int methodId) {
      this.serviceImpl = serviceImpl;
      this.methodId = methodId;
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public void invoke(Req request, io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        case METHODID_REGISTER_MINER:
          serviceImpl.registerMiner((mining.Mining.MinerInfo) request,
              (io.grpc.stub.StreamObserver<mining.Mining.RegistrationResponse>) responseObserver);
          break;
        case METHODID_GET_WORK:
          serviceImpl.getWork((mining.Mining.WorkRequest) request,
              (io.grpc.stub.StreamObserver<mining.Mining.WorkResponse>) responseObserver);
          break;
        case METHODID_SUBMIT_WORK:
          serviceImpl.submitWork((mining.Mining.WorkSubmission) request,
              (io.grpc.stub.StreamObserver<mining.Mining.SubmissionResponse>) responseObserver);
          break;
        case METHODID_HEARTBEAT:
          serviceImpl.heartbeat((mining.Mining.MinerStatus) request,
              (io.grpc.stub.StreamObserver<mining.Mining.HeartbeatResponse>) responseObserver);
          break;
        case METHODID_STOP_MINING:
          serviceImpl.stopMining((mining.Mining.MinerInfo) request,
              (io.grpc.stub.StreamObserver<mining.Mining.StopResponse>) responseObserver);
          break;
        default:
          throw new AssertionError();
      }
    }

    @java.lang.Override
    @java.lang.SuppressWarnings("unchecked")
    public io.grpc.stub.StreamObserver<Req> invoke(
        io.grpc.stub.StreamObserver<Resp> responseObserver) {
      switch (methodId) {
        default:
          throw new AssertionError();
      }
    }
  }

  public static final io.grpc.ServerServiceDefinition bindService(AsyncService service) {
    return io.grpc.ServerServiceDefinition.builder(getServiceDescriptor())
        .addMethod(
          getRegisterMinerMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              mining.Mining.MinerInfo,
              mining.Mining.RegistrationResponse>(
                service, METHODID_REGISTER_MINER)))
        .addMethod(
          getGetWorkMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              mining.Mining.WorkRequest,
              mining.Mining.WorkResponse>(
                service, METHODID_GET_WORK)))
        .addMethod(
          getSubmitWorkMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              mining.Mining.WorkSubmission,
              mining.Mining.SubmissionResponse>(
                service, METHODID_SUBMIT_WORK)))
        .addMethod(
          getHeartbeatMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              mining.Mining.MinerStatus,
              mining.Mining.HeartbeatResponse>(
                service, METHODID_HEARTBEAT)))
        .addMethod(
          getStopMiningMethod(),
          io.grpc.stub.ServerCalls.asyncUnaryCall(
            new MethodHandlers<
              mining.Mining.MinerInfo,
              mining.Mining.StopResponse>(
                service, METHODID_STOP_MINING)))
        .build();
  }

  private static abstract class MiningPoolBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoFileDescriptorSupplier, io.grpc.protobuf.ProtoServiceDescriptorSupplier {
    MiningPoolBaseDescriptorSupplier() {}

    @java.lang.Override
    public com.google.protobuf.Descriptors.FileDescriptor getFileDescriptor() {
      return mining.Mining.getDescriptor();
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.ServiceDescriptor getServiceDescriptor() {
      return getFileDescriptor().findServiceByName("MiningPool");
    }
  }

  private static final class MiningPoolFileDescriptorSupplier
      extends MiningPoolBaseDescriptorSupplier {
    MiningPoolFileDescriptorSupplier() {}
  }

  private static final class MiningPoolMethodDescriptorSupplier
      extends MiningPoolBaseDescriptorSupplier
      implements io.grpc.protobuf.ProtoMethodDescriptorSupplier {
    private final java.lang.String methodName;

    MiningPoolMethodDescriptorSupplier(java.lang.String methodName) {
      this.methodName = methodName;
    }

    @java.lang.Override
    public com.google.protobuf.Descriptors.MethodDescriptor getMethodDescriptor() {
      return getServiceDescriptor().findMethodByName(methodName);
    }
  }

  private static volatile io.grpc.ServiceDescriptor serviceDescriptor;

  public static io.grpc.ServiceDescriptor getServiceDescriptor() {
    io.grpc.ServiceDescriptor result = serviceDescriptor;
    if (result == null) {
      synchronized (MiningPoolGrpc.class) {
        result = serviceDescriptor;
        if (result == null) {
          serviceDescriptor = result = io.grpc.ServiceDescriptor.newBuilder(SERVICE_NAME)
              .setSchemaDescriptor(new MiningPoolFileDescriptorSupplier())
              .addMethod(getRegisterMinerMethod())
              .addMethod(getGetWorkMethod())
              .addMethod(getSubmitWorkMethod())
              .addMethod(getHeartbeatMethod())
              .addMethod(getStopMiningMethod())
              .build();
        }
      }
    }
    return result;
  }
}
