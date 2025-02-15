package auth

// Based on https://github.com/NoMore201/googleplay-api/blob/master/gpapi/googleplay.py

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/Elbandi/go-googleplay/pkg/common"
	"github.com/Elbandi/go-googleplay/pkg/keyring"
	"github.com/Elbandi/go-googleplay/pkg/playstore/pb"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	AuthURL = common.APIBaseURL + "/auth"
	CheckinURL = common.APIBaseURL + "/checkin"
)

/**
Handles authentication transparently
Decides based on config parameters how authentication should be performed
 */
type Client struct {
	config *Config
	deviceConsistencyToken string
}

type Config struct {
	Email string
	Password string
	GsfId string
	AuthSubToken string
}

func CreatePlaystoreAuthClient(config *Config) (*Client, error) {
	if config.GsfId == "" && config.AuthSubToken == "" {
		gsfId, authSub, err := keyring.GetGoogleTokens()
		if err == nil && gsfId != "" && authSub != "" {
			log.Tracef("Found GSIF %s and authSub %s tokens from keyring", gsfId, authSub)
			config.GsfId = gsfId
			config.AuthSubToken = authSub
		}
	}
	return &Client{config: config}, nil
}

type Type string

const (
	EmailPassword Type = "email-pass"
	Token         Type = "token"
	Unknown       Type = ""
)

/**
Use email and passwd if set, otherwise use tokens
 */
func (client *Client) getAuthType() Type {
	if client.config.Email != "" && client.config.Password != "" {
		return EmailPassword
	}

	if client.config.GsfId != ""  && client.config.AuthSubToken  != "" {
		return Token
	}
	return Unknown
}

/**
Check if has necessary tokens (GsfId & AuthSub) for authenticated request, does not check if the tokens are valid
 */
func (client *Client) HasAuthToken() bool {
	return client.config.GsfId != "" && client.config.AuthSubToken != ""
}

func (client *Client) GetGsfId() string {
	return client.config.GsfId
}

func (client *Client) GetAuthSubToken() string {
	return client.config.AuthSubToken
}

type DeviceConfig struct {
	cellOperator string
	simOperator string
	roaming string
}

// Get "androidId", which is a device specific GSF (google services framework) ID
func (client *Client) getGsfId() (string, error) {
	locale := "fi"
	timezone := "Europe/Helsinki"
	version := int32(3)
	fragment := int32(0)

	lastCheckinMsec := int64(0)
	userNumber := int32(0)
	cellOperator := "22210"
	simOperator := "22210"
	roaming := "mobile-notroaming"

	// custom config that should be able to download most apps
	checkin := pb.AndroidCheckinProto{
		Build:           &pb.AndroidBuildProto{
			Id:             stringP("unknown"),
			Product:        stringP("unknown"),
			Carrier:        stringP("unknown"),
			Radio:          stringP("unknown"),
			Bootloader:     stringP("unknown"),
			Client:         stringP("android-google"),
			Timestamp:      int64P(int64(time.Now().Second())),
			GoogleServices: intP(204713063), // Google Play Services Version: 20.47.13
			Device:         stringP("unknown"),
			SdkVersion:     intP(30), // Android 11, the app must support this sdk version
			Model:          stringP("unknown"),
			Manufacturer:   stringP("unknown"),
			BuildProduct:   stringP("unknown"),
			OtaInstalled:   boolP(true),
		},
		LastCheckinMsec: &lastCheckinMsec,
		Event:           nil,
		Stat:            nil,
		RequestedGroup:  nil,
		CellOperator:    &cellOperator,
		SimOperator:     &simOperator,
		Roaming:         &roaming,
		UserNumber:      &userNumber,
	}

	checkinReq := &pb.AndroidCheckinRequest{
		Imei:                nil,
		Id:                  nil,
		Digest:              nil,
		Checkin:             &checkin,
		DesiredBuild:        nil,
		Locale:              &locale,
		LoggingId:           nil,
		MarketCheckin:       nil,
		MacAddr:             nil,
		Meid:                nil,
		AccountCookie:       nil,
		TimeZone:            &timezone,
		SecurityToken:       nil,
		Version:             &version,
		OtaCert:             nil,
		SerialNumber:        nil,
		Esn:                 nil,
		DeviceConfiguration: &pb.DeviceConfigurationProto{
			TouchScreen:            intP(3),
			Keyboard:               intP(2),
			Navigation:             intP(2),
			ScreenLayout:           intP(2),
			HasHardKeyboard:        boolP(true),
			HasFiveWayNavigation:   boolP(true),
			ScreenDensity:          intP(402),
			GlEsVersion:            intP(196610), // OpenGL ES 3.2
			SystemSharedLibrary:    strings.Split("ConnectivityExt,android.ext.services,android.ext.shared,android.hidl.manager@V1.0-java,android.test.mock,android.test.runner,com.android.future.usb.accessory,com.android.location.provider,com.android.media.remotedisplay,com.android.mediadrm.signer,com.dsi.ant.antradio_library,com.google.android.dialer.support,com.google.android.gms,com.google.android.maps,com.google.android.media.effects,com.google.widevine.software.drm,com.qti.dpmapi,com.qti.dpmframework,com.qti.ims.connectionmanager.imscmlibrary,com.qti.location.sdk,com.qti.snapdragon.sdk.display,com.qualcomm.qcnvitems,com.qualcomm.qcrilhook,com.qualcomm.qti.Performance,com.quicinc.cne,com.quicinc.cneapiclient,izat.xt.srv,javax.obex,org.apache.http.legacy,org.lineageos.hardware,org.lineageos.platform,qcom.fmradio", ","),
			SystemAvailableFeature: strings.Split("android.hardware.audio.low_latency,android.hardware.audio.output,android.hardware.bluetooth,android.hardware.bluetooth_le,android.hardware.camera,android.hardware.camera.any,android.hardware.camera.autofocus,android.hardware.camera.capability.manual_post_processing,android.hardware.camera.capability.manual_sensor,android.hardware.camera.capability.raw,android.hardware.camera.flash,android.hardware.camera.front,android.hardware.camera.level.full,android.hardware.consumerir,android.hardware.faketouch,android.hardware.fingerprint,android.hardware.location,android.hardware.location.gps,android.hardware.location.network,android.hardware.microphone,android.hardware.opengles.aep,android.hardware.ram.normal,android.hardware.screen.landscape,android.hardware.screen.portrait,android.hardware.sensor.accelerometer,android.hardware.sensor.compass,android.hardware.sensor.gyroscope,android.hardware.sensor.light,android.hardware.sensor.proximity,android.hardware.sensor.stepcounter,android.hardware.sensor.stepdetector,android.hardware.telephony,android.hardware.telephony.cdma,android.hardware.telephony.gsm,android.hardware.touchscreen,android.hardware.touchscreen.multitouch,android.hardware.touchscreen.multitouch.distinct,android.hardware.touchscreen.multitouch.jazzhand,android.hardware.usb.accessory,android.hardware.usb.host,android.hardware.vulkan.level,android.hardware.vulkan.version,android.hardware.wifi,android.hardware.wifi.direct,android.software.activities_on_secondary_displays,android.software.app_widgets,android.software.autofill,android.software.backup,android.software.companion_device_setup,android.software.connectionservice,android.software.cts,android.software.device_admin,android.software.home_screen,android.software.input_methods,android.software.live_wallpaper,android.software.managed_users,android.software.midi,android.software.picture_in_picture,android.software.print,android.software.sip,android.software.sip.voip,android.software.voice_recognizers,android.software.webview,com.google.android.apps.dialer.SUPPORTED,com.google.android.feature.EXCHANGE_6_2,com.google.android.feature.GOOGLE_BUILD,com.google.android.feature.GOOGLE_EXPERIENCE,org.lineageos.android,org.lineageos.audio,org.lineageos.hardware,org.lineageos.livedisplay,org.lineageos.performance,org.lineageos.profiles,org.lineageos.style,org.lineageos.weather,projekt.substratum.theme", ","),
			NativePlatform:         []string{"armeabi-v7a", "armeabi", "x86", "x86_64", "arm64-v8a"},
			ScreenWidth:            intP(2340),
			ScreenHeight:           intP(1080),
			SystemSupportedLocale:  strings.Split("af,af_ZA,am,am_ET,ar,ar_EG,ar_XB,ast,az,be,bg,bg_BG,bn,bs,ca,ca_ES,cs,cs_CZ,da,da_DK,de,de_AT,de_CH,de_DE,de_LI,el,el_GR,en,en_AU,en_CA,en_GB,en_IN,en_NZ,en_SG,en_US,en_XA,en_XC,eo,es,es_ES,es_US,et,eu,fa,fa_IR,fi,fi_FI,fil,fil_PH,fr,fr_BE,fr_CA,fr_CH,fr_FR,gl,gu,hi,hi_IN,hr,hr_HR,hu,hu_HU,hy,in,in_ID,is,it,it_CH,it_IT,iw,iw_IL,ja,ja_JP,ka,kk,km,kn,ko,ko_KR,ky,lo,lt,lt_LT,lv,lv_LV,mk,ml,mn,mr,ms,ms_MY,my,nb,nb_NO,ne,nl,nl_BE,nl_NL,pa,pl,pl_PL,pt,pt_BR,pt_PT,ro,ro_RO,ru,ru_RU,si,sk,sk_SK,sl,sl_SI,sq,sr,sr_Latn,sr_RS,sv,sv_SE,sw,sw_TZ,ta,te,th,th_TH,tr,tr_TR,uk,uk_UA,ur,uz,vi,vi_VN,zh,zh_CN,zh_HK,zh_TW,zu,zu_ZA", ","),
			GlExtension:            strings.Split(
				"GL_AMD_compressed_ATC_texture,GL_AMD_performance_monitor,GL_ANDROID_extension_pack_es31a,GL_APPLE_texture_2D_limited_npot,GL_ARB_vertex_buffer_object,GL_ARM_shader_framebuffer_fetch_depth_stencil,GL_EXT_EGL_image_array,GL_EXT_YUV_target,GL_EXT_blit_framebuffer_params,GL_EXT_buffer_storage,GL_EXT_clip_cull_distance,GL_EXT_color_buffer_float,GL_EXT_color_buffer_half_float,GL_EXT_copy_image,GL_EXT_debug_label,GL_EXT_debug_marker,GL_EXT_discard_framebuffer,GL_EXT_disjoint_timer_query,GL_EXT_draw_buffers_indexed,GL_EXT_external_buffer,GL_EXT_geometry_shader,GL_EXT_gpu_shader5,GL_EXT_multisampled_render_to_texture,GL_EXT_multisampled_render_to_texture2,GL_EXT_primitive_bounding_box,GL_EXT_protected_textures,GL_EXT_robustness,GL_EXT_sRGB,GL_EXT_sRGB_write_control,GL_EXT_shader_framebuffer_fetch,GL_EXT_shader_io_blocks,GL_EXT_shader_non_constant_global_initializers,GL_EXT_tessellation_shader,GL_EXT_texture_border_clamp,GL_EXT_texture_buffer,GL_EXT_texture_cube_map_array,GL_EXT_texture_filter_anisotropic,GL_EXT_texture_format_BGRA8888,GL_EXT_texture_norm16,GL_EXT_texture_sRGB_R8,GL_EXT_texture_sRGB_decode,GL_EXT_texture_type_2_10_10_10_REV,GL_KHR_blend_equation_advanced,GL_KHR_blend_equation_advanced_coherent,GL_KHR_debug,GL_KHR_no_error,GL_KHR_texture_compression_astc_hdr,GL_KHR_texture_compression_astc_ldr,GL_NV_shader_noperspective_interpolation,GL_OES_EGL_image,GL_OES_EGL_image_external,GL_OES_EGL_image_external_essl3,GL_OES_EGL_sync,GL_OES_blend_equation_separate,GL_OES_blend_func_separate,GL_OES_blend_subtract,GL_OES_compressed_ETC1_RGB8_texture,GL_OES_compressed_paletted_texture,GL_OES_depth24,GL_OES_depth_texture,GL_OES_depth_texture_cube_map,GL_OES_draw_texture,GL_OES_element_index_uint,GL_OES_framebuffer_object,GL_OES_get_program_binary,GL_OES_matrix_palette,GL_OES_packed_depth_stencil,GL_OES_point_size_array,GL_OES_point_sprite,GL_OES_read_format,GL_OES_rgb8_rgba8,GL_OES_sample_shading,GL_OES_sample_variables,GL_OES_shader_image_atomic,GL_OES_shader_multisample_interpolation,GL_OES_standard_derivatives,GL_OES_stencil_wrap,GL_OES_surfaceless_context,GL_OES_texture_3D,GL_OES_texture_compression_astc,GL_OES_texture_cube_map,GL_OES_texture_env_crossbar,GL_OES_texture_float,GL_OES_texture_float_linear,GL_OES_texture_half_float,GL_OES_texture_half_float_linear,GL_OES_texture_mirrored_repeat,GL_OES_texture_npot,GL_OES_texture_stencil8,GL_OES_texture_storage_multisample_2d_array,GL_OES_vertex_array_object,GL_OES_vertex_half_float,GL_OVR_multiview,GL_OVR_multiview2,GL_OVR_multiview_multisampled_render_to_texture,GL_QCOM_alpha_test,GL_QCOM_extended_get,GL_QCOM_framebuffer_foveated,GL_QCOM_shader_framebuffer_fetch_noncoherent,GL_QCOM_tiled_rendering", ","),
			DeviceClass:            nil,
			MaxApkDownloadSizeMb:   intP(100 * 100),
		},
		MacAddrType:         nil,
		Fragment:            &fragment,
		UserName:            nil,
		UserSerialNumber:    nil,
	}


	rawMsg, err := proto.Marshal(checkinReq)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(CheckinURL, "application/x-protobuf", bytes.NewReader(rawMsg))
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var checkinResp pb.AndroidCheckinResponse
	err = proto.Unmarshal(body, &checkinResp)
	if err != nil {
		return "", err
	}

	rawMsg, err = proto.Marshal(checkinReq)
	if err != nil {
		return "", err
	}

	resp, err = http.Post(CheckinURL, "application/x-protobuf", bytes.NewReader(rawMsg))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("checkin error: %s %d", resp.Status, resp.StatusCode)
	}

	return strconv.FormatUint(*checkinResp.AndroidId, 16), nil
}


func (client *Client) Authenticate() error {
	log.Debugf("Authenticate")

	authType := client.getAuthType()
	if authType == Unknown {
		return fmt.Errorf(
			"could not select authentication type. " +
				"Did you specify the email and the password or alternatively GSFID and authSubToken")
	}

	switch authType {
	case EmailPassword:
		encryptedPasswd, err := encryptCredentials(client.config.Email, client.config.Password, nil)
		if err != nil {
			return err
		}

		client.config.GsfId, err = client.getGsfId()
		if err != nil {
			return err
		}

		client.config.AuthSubToken, err = getPlayStoreAuthSubToken(client.config.Email, encryptedPasswd)
		if err != nil {
			return err
		}

		log.Infof("Got GsfId and AuthSubToken, saving these to keyring")

		err = keyring.SaveToken(keyring.GSFID, client.config.GsfId)
		if err != nil {
			return err
		}

		err = keyring.SaveToken(keyring.AuthSubToken, client.config.AuthSubToken)
		if err != nil {
			return err
		}
	}

	return nil
}
